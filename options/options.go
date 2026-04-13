package options

import (
	"log/slog"
	"time"

	"github.com/athoune/fluent-server/msg"
	"github.com/athoune/fluent-server/wire"
)

type FluentOptions struct {
	MessagesReaderConfig  map[string]interface{}
	MessagesReaderFactory MessagesReaderFactory
	SharedKey             string
	Hostname              string
	Logger                *slog.Logger
	Users                 func(string) []byte
	Debug                 bool
	// ReadTimeout is the maximum duration for reading the entire request
	ReadTimeout time.Duration
	// WriteTimeout is the maximum duration before timing out writes
	WriteTimeout time.Duration
	// IdleTimeout is the maximum amount of time to wait for the next request
	IdleTimeout time.Duration
}

type Session struct {
	Logger *slog.Logger
	Reader *FluentReader
}

type FluentReader struct {
	MessagesReaderFactory MessagesReaderFactory
}

type MessagesReaderFactory func(log *slog.Logger, cfg map[string]interface{}) MessagesReader

type MessagesReader interface {
	ForwardMode(wire *wire.Wire, tag string) error
	PackedForwardMode(tag string, blob []byte, opt *msg.Option) error
	MessageMode(wire *wire.Wire, tag string) error
}
