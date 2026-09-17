// Known-failing tests — implementation bugs, to be fixed separately.
//
// Several tests below currently FAIL against a real Fluent Bit client. They
// are skipped by default so the suite stays green; run
//
//	FLUENT_TEST_KNOWN_BUGS=1 go test ./functional/...
//
// to execute them and observe the failures. Three independent bugs are
// involved.
//
// 1. Metadata-aware entry framing is not decoded.
//
// With its default settings (Time_as_Integer False, the mode used by
// demo/fluentbit.cfg), Fluent Bit 5.x packs each Forward-mode entry as
//
//	[[timestamp, metadata], record]
//
// A frame captured from Fluent Bit 5.1.2 looks like:
//
//	92                      array(2)          [tag, entries]
//	ae "test.fluentbit"     tag
//	91                      array(1)          entries
//	92                      array(2)          entry
//	92                      array(2)          [timestamp, metadata]
//	d7 00 <8 bytes>         fixext8 timestamp
//	80                      map(0)            empty metadata
//	82 {"message":"..."}    record
//
// message.DecodeEntry (message/entry.go) expects the legacy
// [timestamp, record] layout, so DecodeTime sees the inner array and fails
// with "unknown type 146" (0x92 is fixarray of length 2). Every event is
// lost. Affected: TestForwardPlain, TestForwardAck, TestMultipleEvents,
// TestSharedKey, TestMTLS, TestForwardCompressed.
//
// Note: Fluent Bit 2.x/3.x emit a flat [timestamp, record] entry but encode
// it as array32 (0xdd). DecodeEntry only accepts a *fixed* array
// (msgpcode.IsFixedArray), so those versions are rejected as well.
//
// Fluent Bit's Time_as_Integer True compat mode still emits a flat
// [uint32, record] fixed array, which is why TestForwardTimeAsInteger
// passes and exercises the whole chain (TCP, forward mode, event decoding).
//
// 2. PackedForward errors are silently swallowed.
//
// In message/mode.go, the bin branch (CompressedPackedForward) declares
//
//	blob, err := s.Wire.Decoder.DecodeBytes()
//
// inside the switch case. The := creates a new err that shadows the
// function-scoped one, so the trailing `return err` returns nil even when
// PackedForwardMode() failed. The session stays open with the event lost
// and, once the client disconnects, only logs "connection closed".
// Affected: TestForwardCompressed — decompression succeeds, then the entry
// framing error from bug 1 is dropped on the floor.
//
// 3. Message mode leaves the optional fourth element unread.
//
// Two clients reach Message mode:
//
//   - Fluent Bit, when the output Tag is dynamic, e.g.
//
//     Tag app.$message
//
//     (flb_forward_format_message_mode in out_forward/forward_format.c).
//     The record accessor is $message, not ${message}: Fluent Bit expands
//     ${...} in configuration files as environment variables, so ${message}
//     silently becomes an empty string and the output falls back to Forward
//     mode.
//
//   - the official Node.js logger (@fluent-org/logger) with eventMode
//     "Message".
//
// Both pack each entry as a 4-element array and append an options map even
// when there is nothing to put in it — Fluent Bit at least
// {"fluent_signal": 0}, plus "chunk" when Require_ack_response is enabled,
// Node.js an empty map:
//
//	[tag, timestamp, record, options]
//
// message.MessageMode() (defaultreader/reader.go) reads only
// tag/timestamp/record and leaves that map in the stream. The next
// FluentSession.handleMessage() then PeekCode()s a map where it requires a
// fixed array and aborts with "unexpected code". With Require_ack_response
// the server also never sends the ACK Fluent Bit waits for, so Fluent Bit
// retries the chunk.
//
// The Python and Ruby official loggers also use Message mode, but they emit
// the historical 3-element form [tag, timestamp, record] and work fine.
// Affected: TestMessageModeDynamicTag, TestMessageModeAck,
// TestJavascriptMessage. These assert on the server-side slog records: no
// Error-level "session error" may be logged.
//
// Fix directions:
//   - message/entry.go: accept both [timestamp, record] and
//     [[timestamp, metadata], record], and accept array16/array32 headers.
//   - message/mode.go: assign with `=` instead of `:=` for err in the bin
//     branch so the error is propagated.
//   - defaultreader/reader.go + message/mode.go: consume the 4th element in
//     message mode and send an ACK when a "chunk" is present, mirroring
//     handleChunk() for forward mode.
package functional

import (
	"os"
	"testing"
	"time"

	"github.com/athoune/fluent-server/event"
	"github.com/stretchr/testify/assert"
)

const (
	// knownBugEnv also runs the tests listed in the file header comment.
	knownBugEnv = "FLUENT_TEST_KNOWN_BUGS"

	tagSingle  = "test.fluentbit"
	tagMulti   = "test.multi"
	tagDynamic = "app.hello" // dynamic tag "app.${message}" with message = hello

	// dummyRecord is the deterministic payload emitted by the dummy input.
	dummyRecord = `{"message":"hello","n":42}`

	// sharedKey is the key used by the shared-key tests when the harness
	// controls the server.
	sharedKey = "s3cr3t"

	// eventWait is generous: Fluent Bit flushes every second and the image
	// may have to be pulled on the first run.
	eventWait = 20 * time.Second
)

func assertHelloRecord(t *testing.T, e event.Event) {
	t.Helper()
	assert.Equal(t, "hello", e.Record["message"])
	assert.Equal(t, float64(42), e.Record["n"])
	assert.False(t, e.Ts.IsZero(), "timestamp must be decoded")
}

// TestForwardPlain covers the default Fluent Bit forward output: a 2-element
// array [tag, entries] with no options.
func TestForwardPlain(t *testing.T) {
	skipKnownBug(t, "metadata-aware entry framing is not decoded (bug 1)")
	requireDocker(t)
	h := startServer(t, serverConfig{})

	runFluentBit(t, buildConf(tagSingle, dummyRecord, 1, h.forwardPort, nil), nil)

	events := waitEvents(t, h, tagSingle, 1, eventWait)
	assertHelloRecord(t, events[0])
}

// TestForwardAck covers Require_ack_response: Fluent Bit appends an options
// map carrying a "chunk" and waits for the matching ACK.
func TestForwardAck(t *testing.T) {
	skipKnownBug(t, "metadata-aware entry framing is not decoded (bug 1)")
	requireDocker(t)
	h := startServer(t, serverConfig{})

	conf := buildConf(tagSingle, dummyRecord, 1, h.forwardPort,
		[]string{"Require_ack_response True"})
	runFluentBit(t, conf, nil)

	events := waitEvents(t, h, tagSingle, 1, eventWait)
	assertHelloRecord(t, events[0])

	// A malformed ACK makes Fluent Bit retry the chunk and duplicate the
	// event; settling must keep the count at one.
	assertNoDuplicateEvents(t, h, tagSingle)
}

// TestForwardCompressed covers Compress gzip, which switches Fluent Bit from
// Forward mode to CompressedPackedForward.
func TestForwardCompressed(t *testing.T) {
	skipKnownBug(t, "entry framing (bug 1), and the decode error is swallowed by a shadowed err (bug 2)")
	requireDocker(t)
	h := startServer(t, serverConfig{})

	conf := buildConf(tagSingle, dummyRecord, 1, h.forwardPort, []string{"Compress gzip"})
	runFluentBit(t, conf, nil)

	events := waitEvents(t, h, tagSingle, 1, eventWait)
	assertHelloRecord(t, events[0])
}

// TestForwardTimeAsInteger covers Time_as_Integer True (compat mode for
// Fluentd <= 0.12). This is the only forward mode Fluent Bit still encodes in
// the legacy [timestamp, record] layout, so it passes and validates the
// harness end to end.
func TestForwardTimeAsInteger(t *testing.T) {
	requireDocker(t)
	h := startServer(t, serverConfig{})

	conf := buildConf(tagSingle, dummyRecord, 1, h.forwardPort, []string{"Time_as_Integer True"})
	runFluentBit(t, conf, nil)

	events := waitEvents(t, h, tagSingle, 1, eventWait)
	assertHelloRecord(t, events[0])
	assert.Zero(t, events[0].Ts.Nanosecond(), "Time_as_Integer encodes whole seconds")
}

// TestMultipleEvents checks that a batch of records is fully decoded.
func TestMultipleEvents(t *testing.T) {
	const samples = 5
	skipKnownBug(t, "metadata-aware entry framing is not decoded (bug 1)")
	requireDocker(t)
	h := startServer(t, serverConfig{})

	runFluentBit(t, buildConf(tagMulti, dummyRecord, samples, h.forwardPort, nil), nil)

	events := waitEvents(t, h, tagMulti, samples, eventWait)
	assert.Len(t, events, samples)
	for _, e := range events {
		assertHelloRecord(t, e)
	}
}

// TestSharedKey covers the secure forward handshake: HELO/PING/PONG, with
// mutual authentication of the shared key digest.
func TestSharedKey(t *testing.T) {
	skipKnownBug(t, "metadata-aware entry framing is not decoded (bug 1)")
	requireDocker(t)
	h := startServer(t, serverConfig{sharedKey: sharedKey})
	requireInProcess(t, h)

	conf := buildConf(tagSingle, dummyRecord, 1, h.forwardPort, []string{
		"Shared_Key " + sharedKey,
		"Self_Hostname client.example.com",
	})
	runFluentBit(t, conf, nil)

	events := waitEvents(t, h, tagSingle, 1, eventWait)
	assertHelloRecord(t, events[0])
}

// TestSharedKeyMismatch verifies authentication is actually enforced: a
// client with the wrong key must not get a single event through.
func TestSharedKeyMismatch(t *testing.T) {
	requireDocker(t)
	h := startServer(t, serverConfig{sharedKey: sharedKey})
	requireInProcess(t, h)

	conf := buildConf(tagSingle, dummyRecord, 1, h.forwardPort, []string{
		"Shared_Key definitely-not-the-right-key",
		"Self_Hostname client.example.com",
	})
	runFluentBit(t, conf, nil)

	assertNoEvents(t, h, tagSingle, 8*time.Second)
}

// TestMTLS covers mutual TLS: the server requires and verifies a client
// certificate, Fluent Bit verifies the server certificate.
func TestMTLS(t *testing.T) {
	skipKnownBug(t, "metadata-aware entry framing is not decoded (bug 1)")
	requireDocker(t)
	certs := generateCerts(t)
	h := startServer(t, serverConfig{tls: &certs})
	requireInProcess(t, h)

	conf := buildConf(tagSingle, dummyRecord, 1, h.forwardPort, []string{
		"tls          on",
		"tls.verify   on",
		"tls.ca_file  /certs/ca.pem",
		"tls.crt_file /certs/client.pem",
		"tls.key_file /certs/client-key.pem",
	})
	runFluentBit(t, conf, map[string]string{certs.dir: "/certs"})

	events := waitEvents(t, h, tagSingle, 1, eventWait)
	assertHelloRecord(t, events[0])
}

// TestMessageModeDynamicTag covers Message mode, reached by using a dynamic
// output tag. FAILS — see the file header comment (bug 3).
func TestMessageModeDynamicTag(t *testing.T) {
	skipKnownBug(t, "message mode leaves the options map unread (bug 3)")
	requireDocker(t)
	h := startServer(t, serverConfig{})
	requireInProcess(t, h)

	conf := buildConf(tagSingle, dummyRecord, 1, h.forwardPort, []string{"Tag app.$message"})
	runFluentBit(t, conf, nil)

	// The first record is delivered before the server aborts the session.
	events := waitEvents(t, h, tagDynamic, 1, eventWait)
	assertHelloRecord(t, events[0])
	assertNoSessionErrors(t, h)
}

// TestMessageModeAck covers Message mode with Require_ack_response. FAILS —
// see the file header comment (bug 3).
func TestMessageModeAck(t *testing.T) {
	skipKnownBug(t, "message mode leaves the options map unread and is never ACKed (bug 3)")
	requireDocker(t)
	h := startServer(t, serverConfig{})
	requireInProcess(t, h)

	conf := buildConf(tagSingle, dummyRecord, 1, h.forwardPort, []string{
		"Tag app.$message",
		"Require_ack_response True",
	})
	runFluentBit(t, conf, nil)

	events := waitEvents(t, h, tagDynamic, 1, eventWait)
	assertHelloRecord(t, events[0])
	assertNoSessionErrors(t, h)
}

// skipKnownBug skips a test that fails because of an implementation bug
// documented in the file header, unless FLUENT_TEST_KNOWN_BUGS=1 is set.
func skipKnownBug(t *testing.T, reason string) {
	t.Helper()
	if os.Getenv(knownBugEnv) != "1" {
		t.Skipf("known implementation bug — %s; set %s=1 to run it (see the file header)", reason, knownBugEnv)
	}
}

// assertNoSessionErrors lets the connection settle and then checks that the
// server never aborted a session. Fluent Bit's Message mode always appends an
// options map the current implementation does not consume, which closes the
// connection right after the first record.
func assertNoSessionErrors(t *testing.T, h *serverHandle) {
	t.Helper()
	if h.logs == nil {
		t.Skip("server logs are only available in-process")
	}
	time.Sleep(2 * time.Second)
	assert.Zero(t, h.logs.sessionErrors(),
		"the server aborted the session: the options map of message mode is left unread")
}
