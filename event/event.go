package event

// FIXME unify event.Event and message.Event

import (
	"time"
)

type Event struct {
	Ts     time.Time              `json:"ts"`
	Record map[string]interface{} `json:"record"`
}

func New(ts time.Time, record map[string]interface{}) Event {
	return Event{
		Ts:     ts,
		Record: record,
	}
}

type Events []Event

func (e Events) Len() int {
	return len(e)
}

func (e Events) Less(i, j int) bool {
	return e[i].Ts.After(e[j].Ts)
}

func (e Events) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
