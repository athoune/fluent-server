package functional

/*
Functional tests: drive the server with a real Fluent Bit client running in a
container, and assert what the server received through its HTTP mirror.

Two server modes are supported:

  - in-process (default): the harness starts the very same components main.go
    wires together (mirror + server) on ephemeral ports. A single `go test`
    is enough, and breakpoints in server/, message/ and wire/ are hit while
    debugging the test.

  - external: when FLUENT_TEST_MIRROR_URL is set, the harness starts nothing
    and talks to an already running server, e.g. the one you launched from
    VSCode. The forward port is read from FLUENT_TEST_FORWARD_PORT (default
    24224), and the server must listen on 0.0.0.0 so the container can reach
    it. Tests that need control over the server configuration or logs are
    skipped in this mode.
*/

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/athoune/fluent-server/defaultreader"
	"github.com/athoune/fluent-server/event"
	"github.com/athoune/fluent-server/mirror"
	"github.com/athoune/fluent-server/options"
	"github.com/athoune/fluent-server/server"
	"github.com/stretchr/testify/assert"
)

const (
	// fluentBitImage is pinned so test results do not drift with upstream
	// releases. The official image is multi-arch (amd64/arm64/armv7).
	fluentBitImage = "fluent/fluent-bit:5.1.2"

	// hostFromContainer is how the Fluent Bit container reaches the host.
	// host-gateway makes it work on Linux too, where the name is not
	// resolved by the Docker daemon.
	hostFromContainer = "host.docker.internal"

	defaultForwardPort = 24224
)

var mirrorClient = &http.Client{Timeout: 5 * time.Second}

func TestMain(m *testing.M) {
	// The mirror logs every event at Info level through the default logger.
	// Silence it so the test output stays readable.
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

//
// Docker
//

var (
	dockerOnce sync.Once
	dockerErr  error
)

// requireDocker skips the test when no usable Docker daemon is available, so
// that `go test ./...` stays runnable on a machine without Docker.
func requireDocker(t *testing.T) {
	t.Helper()
	dockerOnce.Do(func() {
		if _, err := exec.LookPath("docker"); err != nil {
			dockerErr = fmt.Errorf("docker not found in PATH: %w", err)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if out, err := exec.CommandContext(ctx, "docker", "info").CombinedOutput(); err != nil {
			dockerErr = fmt.Errorf("docker daemon unavailable: %w: %s", err, strings.TrimSpace(string(out)))
		}
	})
	if dockerErr != nil {
		t.Skipf("skipping functional test: %v", dockerErr)
	}
}

// runFluentBit writes conf to a temporary directory, starts the Fluent Bit
// container against the local server and returns the container name. The
// container is removed when the test ends; its logs are dumped on failure.
func runFluentBit(t *testing.T, conf string, mounts map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fluent-bit.conf"), []byte(conf), 0o644); err != nil {
		t.Fatalf("write fluent-bit.conf: %v", err)
	}

	name := "fluent-server-test-" + randomSuffix(t)
	args := []string{
		"run", "-d",
		"--name", name,
		"--add-host", hostFromContainer + ":host-gateway",
		"-v", dir + ":/cfg:ro",
	}
	for host, target := range mounts {
		args = append(args, "-v", host+":"+target+":ro")
	}
	// The image entrypoint already is /fluent-bit/bin/fluent-bit; the
	// container is distroless, so arguments are passed directly.
	args = append(args, fluentBitImage, "-c", "/cfg/fluent-bit.conf")

	if out, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
		t.Fatalf("docker run failed: %v\n%s", err, out)
	}

	t.Cleanup(func() {
		if t.Failed() {
			if logs, err := exec.Command("docker", "logs", name).CombinedOutput(); err == nil {
				t.Logf("fluent-bit logs (container %s):\n%s", name, logs)
			}
		}
		_ = exec.Command("docker", "rm", "-f", name).Run()
	})

	return name
}

func randomSuffix(t *testing.T) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("generate random suffix: %v", err)
	}
	return hex.EncodeToString(b)
}

// officialClientImage builds, and returns the tag of, an official Fluent
// client image defined in functional/testdata. Docker caches the layers, so
// only the first build of a run does real work.
func officialClientImage(t *testing.T, name string) string {
	t.Helper()
	tag := "fluent-server-test/" + name + ":local"
	cmd := exec.Command("docker", "build", "-q",
		"-t", tag,
		"-f", filepath.Join("testdata", name+".Dockerfile"),
		"testdata",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("docker build %s: %v\n%s", name, err, out)
	}
	return tag
}

// runOfficialClient runs an official Fluent client container against the
// server under test, and fails the test if the client exits non-zero. The
// container reaches the host through host.docker.internal, so the server must
// listen on all interfaces.
func runOfficialClient(t *testing.T, image string, env map[string]string) {
	t.Helper()
	args := []string{"run", "--rm", "--add-host", hostFromContainer + ":host-gateway"}
	for key, value := range env {
		args = append(args, "-e", key+"="+value)
	}
	args = append(args, image)
	if out, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
		t.Fatalf("client %s failed: %v\n%s", image, err, out)
	}
}

// clientEnv assembles the environment shared by every official client.
func clientEnv(h *serverHandle, tag string) map[string]string {
	return map[string]string{
		"FLUENT_HOST": hostFromContainer,
		"FLUENT_PORT": strconv.Itoa(h.forwardPort),
		"FLUENT_TAG":  tag,
	}
}

// assertOfficialRecord checks the record emitted by the testdata clients.
func assertOfficialRecord(t *testing.T, e event.Event, client string) {
	t.Helper()
	assert.Equal(t, client, e.Record["client"])
	assert.Equal(t, float64(42), e.Record["value"])
}

// buildConf renders a minimal Fluent Bit configuration: one dummy input that
// emits a deterministic record, and one forward output pointing at the host.
func buildConf(inputTag, dummy string, samples, port int, outputOptions []string) string {
	var b strings.Builder
	b.WriteString("[SERVICE]\n    Flush 1\n\n")
	b.WriteString("[INPUT]\n    Name    dummy\n")
	fmt.Fprintf(&b, "    Tag     %s\n", inputTag)
	fmt.Fprintf(&b, "    Dummy   %s\n", dummy)
	fmt.Fprintf(&b, "    Samples %d\n", samples)
	b.WriteString("\n[OUTPUT]\n    Name    forward\n    Match   *\n")
	fmt.Fprintf(&b, "    Host    %s\n    Port    %d\n", hostFromContainer, port)
	for _, opt := range outputOptions {
		fmt.Fprintf(&b, "    %s\n", opt)
	}
	return b.String()
}

//
// Server
//

type serverConfig struct {
	sharedKey string
	tls       *tlsMaterial // nil when TLS is disabled
}

type serverHandle struct {
	mirrorURL   string
	forwardPort int
	external    bool
	logs        *recordingHandler // nil in external mode
}

// startServer brings up a server, either in-process or by pointing at an
// externally managed one. See the package comment for the two modes.
func startServer(t *testing.T, cfg serverConfig) *serverHandle {
	t.Helper()

	if url := os.Getenv("FLUENT_TEST_MIRROR_URL"); url != "" {
		port := defaultForwardPort
		if v := os.Getenv("FLUENT_TEST_FORWARD_PORT"); v != "" {
			p, err := strconv.Atoi(v)
			if err != nil {
				t.Fatalf("invalid FLUENT_TEST_FORWARD_PORT %q: %v", v, err)
			}
			port = p
		}
		return &serverHandle{mirrorURL: url, forwardPort: port, external: true}
	}

	mirrorPort := freePort(t)
	forwardPort := freePort(t)

	// The mirror handler is what the assertions read.
	m := mirror.New()
	logs := &recordingHandler{}
	opts := &options.FluentOptions{
		Logger:                slog.New(logs),
		SharedKey:             cfg.sharedKey,
		Debug:                 true,
		MessagesReaderFactory: defaultreader.DefaultMessagesReaderFactory(m.Handler),
	}

	mirrorListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", mirrorPort))
	if err != nil {
		t.Fatalf("mirror listen: %v", err)
	}
	mirrorSrv := &http.Server{Handler: m}
	go func() { _ = mirrorSrv.Serve(mirrorListener) }()

	var s *server.Server
	if cfg.tls != nil {
		tlsCfg, err := server.ConfigTLS(cfg.tls.caFile, cfg.tls.serverCert, cfg.tls.serverKey)
		if err != nil {
			t.Fatalf("server.ConfigTLS: %v", err)
		}
		s, err = server.NewTLS(opts, tlsCfg)
		if err != nil {
			t.Fatalf("server.NewTLS: %v", err)
		}
	} else {
		s, err = server.New(opts)
		if err != nil {
			t.Fatalf("server.New: %v", err)
		}
	}

	// The server must listen on all interfaces: on Linux a socket bound to
	// 127.0.0.1 is not reachable from the Docker bridge.
	go func() { _ = s.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", forwardPort)) }()
	waitForTCP(t, forwardPort, 5*time.Second)

	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("server log records:\n%s", logs.dump())
		}
		_ = s.Shutdown()
		_ = mirrorSrv.Close()
	})

	return &serverHandle{
		mirrorURL:   fmt.Sprintf("http://127.0.0.1:%d", mirrorPort),
		forwardPort: forwardPort,
		logs:        logs,
	}
}

// requireInProcess skips a test that cannot run against an externally managed
// server (one whose configuration and logs the harness does not control).
func requireInProcess(t *testing.T, h *serverHandle) {
	t.Helper()
	if h.external {
		t.Skip("test needs control over the server configuration or logs; unset FLUENT_TEST_MIRROR_URL to run it")
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate a free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitForTCP(t *testing.T, port int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server did not listen on port %d within %s", port, timeout)
}

//
// Assertions
//

// fetchEvents returns the mirror content, or nil on a transient error.
func fetchEvents(t *testing.T, h *serverHandle) map[string]event.Events {
	t.Helper()
	resp, err := mirrorClient.Get(h.mirrorURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var all map[string]event.Events
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return nil
	}
	return all
}

// waitEvents polls the mirror until want events are recorded under tag.
func waitEvents(t *testing.T, h *serverHandle, tag string, want int, timeout time.Duration) event.Events {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		all := fetchEvents(t, h)
		if len(all[tag]) >= want {
			return all[tag]
		}
		if time.Now().After(deadline) {
			body, err := rawMirrorBody(h)
			t.Fatalf("timed out after %s waiting for %d event(s) tagged %q; mirror holds tags %v (raw body %q, err %v)",
				timeout, want, tag, tagList(all), body, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// assertNoEvents waits for the whole window and fails if anything showed up.
func assertNoEvents(t *testing.T, h *serverHandle, tag string, window time.Duration) {
	t.Helper()
	time.Sleep(window)
	all := fetchEvents(t, h)
	assert.Empty(t, all[tag], "expected no event tagged %q", tag)
}

// assertNoDuplicateEvents lets the connection settle, then checks that the
// event was not retried. A retry means the acknowledgement was missing or
// malformed.
func assertNoDuplicateEvents(t *testing.T, h *serverHandle, tag string) {
	t.Helper()
	time.Sleep(2 * time.Second)
	assert.Len(t, fetchEvents(t, h)[tag], 1, "a missing or malformed ACK makes the client retry")
}

func tagList(all map[string]event.Events) []string {
	tags := make([]string, 0, len(all))
	for tag := range all {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// rawMirrorBody is a diagnostic helper: it reports what the mirror actually
// answered when a wait timed out.
func rawMirrorBody(h *serverHandle) (string, error) {
	resp, err := mirrorClient.Get(h.mirrorURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return string(body), err
}

//
// Log capture
//

// recordingHandler captures slog records so tests can assert on server
// behaviour that is otherwise invisible (aborted sessions, for instance).
type recordingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(string) slog.Handler      { return h }

// sessionErrors counts the Error-level "session error" records the server
// emits in message/session.go whenever a session is aborted.
func (h *recordingHandler) sessionErrors() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, r := range h.records {
		if r.Level >= slog.LevelError && r.Message == "session error" {
			n++
		}
	}
	return n
}

// dump renders every captured record, for failure diagnostics.
func (h *recordingHandler) dump() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var b strings.Builder
	for _, r := range h.records {
		fmt.Fprintf(&b, "%s: %s", r.Level, r.Message)
		r.Attrs(func(a slog.Attr) bool {
			fmt.Fprintf(&b, " %s=%v", a.Key, a.Value)
			return true
		})
		b.WriteString("\n")
	}
	return b.String()
}

//
// TLS material
//

type tlsMaterial struct {
	dir        string
	caFile     string
	serverCert string
	serverKey  string
	clientCert string
	clientKey  string
}

// generateCerts issues a throwaway CA, a server certificate valid for
// host.docker.internal (the name Fluent Bit connects to, so tls.verify on
// succeeds) and a client certificate.
func generateCerts(t *testing.T) tlsMaterial {
	t.Helper()
	dir := t.TempDir()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "fluent-server functional test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create CA certificate: %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse CA certificate: %v", err)
	}

	serverDER, serverKey := issueCert(t, caCert, caKey, "host.docker.internal",
		[]string{hostFromContainer, "localhost"}, []net.IP{net.ParseIP("127.0.0.1")},
		x509.ExtKeyUsageServerAuth, 2)
	clientDER, clientKey := issueCert(t, caCert, caKey, "fluent-bit-client",
		nil, nil, x509.ExtKeyUsageClientAuth, 3)

	mat := tlsMaterial{
		dir:        dir,
		caFile:     filepath.Join(dir, "ca.pem"),
		serverCert: filepath.Join(dir, "server.pem"),
		serverKey:  filepath.Join(dir, "server-key.pem"),
		clientCert: filepath.Join(dir, "client.pem"),
		clientKey:  filepath.Join(dir, "client-key.pem"),
	}
	writePEM(t, mat.caFile, "CERTIFICATE", caDER)
	writePEM(t, mat.serverCert, "CERTIFICATE", serverDER)
	writePEM(t, mat.serverKey, "PRIVATE KEY", serverKey)
	writePEM(t, mat.clientCert, "CERTIFICATE", clientDER)
	writePEM(t, mat.clientKey, "PRIVATE KEY", clientKey)
	return mat
}

func issueCert(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey, cn string,
	dnsNames []string, ips []net.IP, usage x509.ExtKeyUsage, serial int64) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key for %s: %v", cn, err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{usage},
		DNSNames:     dnsNames,
		IPAddresses:  ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create certificate for %s: %v", cn, err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key for %s: %v", cn, err)
	}
	return der, keyDER
}

func writePEM(t *testing.T, path, blockType string, der []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: blockType, Bytes: der}); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
}
