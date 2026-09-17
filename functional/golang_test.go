package functional

import (
	"testing"
	"time"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/stretchr/testify/assert"
)

const tagGoClient = "testgo"

// goClient builds an official Go logger (fluent-logger-golang) wired to the
// server under test. The Go client speaks plain TCP, so it needs no container
// and runs even without Docker.
func goClient(t *testing.T, h *serverHandle, config fluent.Config) *fluent.Fluent {
	t.Helper()
	config.FluentHost = "127.0.0.1"
	config.FluentPort = h.forwardPort
	if config.Timeout == 0 {
		config.Timeout = 3 * time.Second
	}
	logger, err := fluent.New(config)
	if err != nil {
		t.Fatalf("fluent.New: %v", err)
	}
	t.Cleanup(func() { _ = logger.Close() })
	return logger
}

func postGoEvent(t *testing.T, logger *fluent.Fluent) {
	t.Helper()
	if err := logger.Post(tagGoClient+".event", map[string]interface{}{"client": "go", "value": 42}); err != nil {
		t.Fatalf("fluent.Post: %v", err)
	}
}

// TestGoClientPost covers the official Go logger in its default, historical
// format: an integer epoch timestamp.
func TestGoClientPost(t *testing.T) {
	h := startServer(t, serverConfig{})

	postGoEvent(t, goClient(t, h, fluent.Config{}))

	events := waitEvents(t, h, tagGoClient+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "go")
	assert.Zero(t, events[0].Ts.Nanosecond(), "the default Go logger uses whole-second epoch time")
}

// TestGoClientSubSecondPrecision covers the contemporary timestamp format:
// SubSecondPrecision switches the logger to EventTime ext.
func TestGoClientSubSecondPrecision(t *testing.T) {
	h := startServer(t, serverConfig{})

	postGoEvent(t, goClient(t, h, fluent.Config{SubSecondPrecision: true}))

	events := waitEvents(t, h, tagGoClient+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "go")
	assert.NotZero(t, events[0].Ts.Nanosecond(), "SubSecondPrecision uses EventTime ext")
}

// TestGoClientRequestAck covers acknowledgements: the logger sends a chunk id
// and waits for the server to acknowledge it. FAILS — see the file header
// comment of fluentbit_test.go: the option map carrying the chunk is the
// fourth element of the Message mode packet, and it is not consumed.
func TestGoClientRequestAck(t *testing.T) {
	skipKnownBug(t, "message mode leaves the options map unread (bug 3)")
	h := startServer(t, serverConfig{})

	postGoEvent(t, goClient(t, h, fluent.Config{RequestAck: true}))

	events := waitEvents(t, h, tagGoClient+".event", 1, eventWait)
	assertOfficialRecord(t, events[0], "go")
	assertNoDuplicateEvents(t, h, tagGoClient+".event")
}
