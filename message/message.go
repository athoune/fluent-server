package message

/*
message module implements the fluentd protocol, using msgpack
*/

import (
	"time"
)

type Event struct {
	tag    string
	ts     *time.Time
	record map[string]interface{}
}

// HandlerFunc handles an event
type HandlerFunc func(tag string, time *time.Time, record map[string]interface{}) error

type FluentReader struct {
	eventHandler HandlerFunc
}

func New(eventHandler HandlerFunc) *FluentReader {
	return &FluentReader{
		eventHandler: eventHandler,
	}
}
