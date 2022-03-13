package message

import (
	"fmt"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

func TestMode(t *testing.T) {
	for _, m := range []struct {
		size    int
		message interface{}
	}{
		{
			size: 2,
			message: []interface{}{
				"tag.name",
				[]interface{}{
					[]interface{}{1441588984, map[string]interface{}{
						"message": "foo",
					}},
					[]interface{}{1441588985, map[string]interface{}{
						"message": "bar",
					}},
				},
				map[string]interface{}{
					"chunk": "p8n9gmxTQVC8/nh2wlKKeQ==",
				},
			},
		},
		{
			size: 1,
			message: []interface{}{
				"tag.name",
				1441588984,
				map[string]interface{}{
					"message": "baz",
				},
				map[string]interface{}{},
			},
		},
	} {
		wg := &sync.WaitGroup{}
		addr, err := mockupServer(&FluentSession{
			Logger: log.Default(),
			Reader: &FluentReader{
				eventHandler: func(tag string, time *time.Time, record map[string]interface{}) error {
					fmt.Println(tag, time, record)
					wg.Done()
					return nil
				},
			},
			step: WaitingForEvents,
		})
		assert.NoError(t, err)
		wg.Add(m.size)
		client, err := net.Dial("tcp", addr.String())
		assert.NoError(t, err)
		encoder := msgpack.NewEncoder(client)
		err = encoder.Encode(m.message)
		assert.NoError(t, err)
		wg.Wait()
		client.Close()
	}
}
