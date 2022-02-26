package mirror

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/factorysh/fluent-server/event"
)

type Mirror struct {
	lock   *sync.Mutex
	events map[string]event.Events
}

func New() *Mirror {
	return &Mirror{
		lock:   &sync.Mutex{},
		events: make(map[string]event.Events),
	}
}

func (t *Mirror) Handler(tag string, ts *time.Time, record map[string]interface{}) error {
	fmt.Println(tag, ts, record)
	t.lock.Lock()
	defer t.lock.Unlock()
	evts, ok := t.events[tag]
	if !ok {
		t.events[tag] = event.Events{
			event.New(*ts, record),
		}
	} else {
		t.events[tag] = append(evts, event.New(*ts, record))
	}
	return nil
}

func (t *Mirror) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(t.events)
	if err != nil {
		panic(err)
	}
}
