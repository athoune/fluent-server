package message

import (
	"time"
)

type Event struct {
	tag    string
	ts     *time.Time
	record map[string]interface{}
}

type HandlerFunc func(tag string, time *time.Time, record map[string]interface{}) error

type FluentReader struct {
	eventHandler HandlerFunc
}

func New(eventHandler HandlerFunc) *FluentReader {
	return &FluentReader{
		eventHandler: eventHandler,
	}
}
