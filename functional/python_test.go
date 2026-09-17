package functional

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const tagPython = "testpython"

// TestPythonMessageHistorical covers the official Python logger in its
// default, historical format: Message mode with an integer epoch timestamp.
func TestPythonMessageHistorical(t *testing.T) {
	requireDocker(t)
	h := startServer(t, serverConfig{})
	image := officialClientImage(t, "python")

	runOfficialClient(t, image, clientEnv(h, tagPython))

	events := waitEvents(t, h, tagPython+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "python")
	assert.Zero(t, events[0].Ts.Nanosecond(), "the default Python logger uses whole-second epoch time")
}

// TestPythonMessageEventTime covers the contemporary timestamp format:
// nanosecond_precision switches the logger to EventTime ext.
func TestPythonMessageEventTime(t *testing.T) {
	requireDocker(t)
	h := startServer(t, serverConfig{})
	image := officialClientImage(t, "python")

	env := clientEnv(h, tagPython)
	env["FLUENT_NANOSECOND_PRECISION"] = "1"
	runOfficialClient(t, image, env)

	events := waitEvents(t, h, tagPython+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "python")
	assert.NotZero(t, events[0].Ts.Nanosecond(), "nanosecond_precision uses EventTime ext")
}
