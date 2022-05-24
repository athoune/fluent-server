package message

/*
message module implements the fluentd protocol, using msgpack
*/

import (
	"time"
)

type Event struct {
	Tag    string
	Ts     *time.Time
	Record map[string]interface{}
}
