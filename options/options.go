package options

import (
	"log"

	"github.com/athoune/fluent-server/msg"
	"github.com/athoune/fluent-server/wire"
)

type FluentOptions struct {
	MessagesReaderConfig  map[string]interface{}
	MessagesReaderFactory MessagesReaderFactory
	SharedKey             string
	Hostname              string
	Logger                *log.Logger
	Users                 func(string) []byte
	Debug                 bool
}

type Session struct {
	Logger *log.Logger
	Reader *FluentReader
}

type FluentReader struct {
	MessagesReaderFactory MessagesReaderFactory
}

type MessagesReaderFactory func(log *log.Logger, cfg map[string]interface{}) MessagesReader

type MessagesReader interface {
	ForwardMode(wire *wire.Wire, tag string) error
	PackedForwardMode(tag string, blob []byte, opt *msg.Option) error
	MessageMode(wire *wire.Wire, tag string) error
}
