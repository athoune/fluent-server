package functional

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const tagRuby = "testruby"

// TestRubyMessageHistorical covers the official Ruby logger in its default,
// historical format: Message mode with an integer epoch timestamp.
func TestRubyMessageHistorical(t *testing.T) {
	requireDocker(t)
	h := startServer(t, serverConfig{})
	image := officialClientImage(t, "ruby")

	runOfficialClient(t, image, clientEnv(h, tagRuby))

	events := waitEvents(t, h, tagRuby+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "ruby")
	assert.Zero(t, events[0].Ts.Nanosecond(), "the default Ruby logger uses whole-second epoch time")
}

// TestRubyMessageEventTime covers the contemporary timestamp format:
// nanosecond_precision switches the logger to EventTime ext.
func TestRubyMessageEventTime(t *testing.T) {
	requireDocker(t)
	h := startServer(t, serverConfig{})
	image := officialClientImage(t, "ruby")

	env := clientEnv(h, tagRuby)
	env["FLUENT_NANOSECOND_PRECISION"] = "1"
	runOfficialClient(t, image, env)

	events := waitEvents(t, h, tagRuby+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "ruby")
	assert.NotZero(t, events[0].Ts.Nanosecond(), "nanosecond_precision uses EventTime ext")
}
