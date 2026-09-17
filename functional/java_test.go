package functional

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const tagJava = "testjava"

// TestJavaClientMessage covers the official Java logger. It uses Message mode with
// the historical integer epoch timestamp: fluent-logger-java does not expose
// the EventTime ext format.
func TestJavaClientMessage(t *testing.T) {
	requireDocker(t)
	h := startServer(t, serverConfig{})
	image := officialClientImage(t, "java")

	runOfficialClient(t, image, clientEnv(h, tagJava))

	events := waitEvents(t, h, tagJava+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "java")
	assert.Zero(t, events[0].Ts.Nanosecond(), "fluent-logger-java uses whole-second epoch time")
}
