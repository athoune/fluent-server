package functional

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	tagJavascript = "testjavascript"
	sharedKeyJS   = "s3cr3t"
)

// jsEnv builds the environment for the Node.js client.
func jsEnv(h *serverHandle, mode string) map[string]string {
	env := clientEnv(h, tagJavascript)
	env["FLUENT_EVENT_MODE"] = mode
	return env
}

// TestJavascriptForward covers the Forward event mode.
func TestJavascriptForward(t *testing.T) {
	requireDocker(t)
	h := startServer(t, serverConfig{})
	image := officialClientImage(t, "javascript")

	runOfficialClient(t, image, jsEnv(h, "Forward"))

	events := waitEvents(t, h, tagJavascript+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "javascript")
}

// TestJavascriptPackedForward covers the default event mode of the Node.js
// client, with acknowledgements enabled.
func TestJavascriptPackedForwardAck(t *testing.T) {
	requireDocker(t)
	h := startServer(t, serverConfig{})
	image := officialClientImage(t, "javascript")

	env := jsEnv(h, "PackedForward")
	env["FLUENT_ACK"] = "1"
	runOfficialClient(t, image, env)

	events := waitEvents(t, h, tagJavascript+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "javascript")
	assertNoDuplicateEvents(t, h, tagJavascript+".event")
}

// TestJavascriptCompressedPackedForward covers the CompressedPackedForward
// event mode. It exercises the PackedForward decoding path of the server.
func TestJavascriptCompressedPackedForward(t *testing.T) {
	requireDocker(t)
	h := startServer(t, serverConfig{})
	image := officialClientImage(t, "javascript")

	runOfficialClient(t, image, jsEnv(h, "CompressedPackedForward"))

	events := waitEvents(t, h, tagJavascript+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "javascript")
}

// TestJavascriptEventTime covers the contemporary timestamp format: the
// EventTime ext type, with explicit seconds and nanoseconds.
func TestJavascriptEventTime(t *testing.T) {
	requireDocker(t)
	h := startServer(t, serverConfig{})
	image := officialClientImage(t, "javascript")

	env := jsEnv(h, "PackedForward")
	env["FLUENT_EVENT_TIME"] = "1"
	runOfficialClient(t, image, env)

	events := waitEvents(t, h, tagJavascript+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "javascript")
	assert.Equal(t, 123456789, events[0].Ts.Nanosecond(), "EventTime carries nanoseconds")
}

// TestJavascriptSharedKey covers shared key authentication with the Node.js
// client, which performs the full HELO/PING/PONG handshake.
func TestJavascriptSharedKey(t *testing.T) {
	requireDocker(t)
	h := startServer(t, serverConfig{sharedKey: sharedKeyJS})
	requireInProcess(t, h)
	image := officialClientImage(t, "javascript")

	env := jsEnv(h, "PackedForward")
	env["FLUENT_ACK"] = "1"
	env["FLUENT_SHARED_KEY"] = sharedKeyJS
	runOfficialClient(t, image, env)

	events := waitEvents(t, h, tagJavascript+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "javascript")
}

// TestJavascriptMessage covers the Message event mode. FAILS — see the file
// header comment of fluentbit_test.go: the client always appends an options
// map (even an empty one), which message mode does not consume.
func TestJavascriptMessage(t *testing.T) {
	skipKnownBug(t, "message mode leaves the options map unread (bug 3)")
	requireDocker(t)
	h := startServer(t, serverConfig{})
	requireInProcess(t, h)
	image := officialClientImage(t, "javascript")

	runOfficialClient(t, image, jsEnv(h, "Message"))

	// The record is delivered before the server aborts the session.
	events := waitEvents(t, h, tagJavascript+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "javascript")
	assertNoSessionErrors(t, h)
}
