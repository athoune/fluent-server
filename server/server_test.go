package server

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/athoune/fluent-server/defaultreader"
	"github.com/athoune/fluent-server/options"
	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

func TestServer(t *testing.T) {
	wg := &sync.WaitGroup{}

	config := &options.FluentOptions{
		MessagesReaderFactory: defaultreader.DefaultMessagesReaderFactory(func(tag string, time *time.Time, record map[string]interface{}) error {
			wg.Done()
			return nil
		}),
	}
	server, err := New(config)
	assert.NoError(t, err)
	server.useUDP = false
	go server.ListenAndServe("127.0.0.1:0")
	server.waitListen.Wait()
	client, err := net.Dial("tcp", server.listener.Addr().String())
	assert.NoError(t, err)
	encoder := msgpack.NewEncoder(client)
	decoder := msgpack.NewDecoder(client)

	wg.Add(2)
	err = encoder.Encode([]interface{}{
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
			"size":  1,
		},
	})
	assert.NoError(t, err)

	m, err := decoder.DecodeMap()
	assert.NoError(t, err)
	assert.Equal(t, "p8n9gmxTQVC8/nh2wlKKeQ==", m["ack"])
	wg.Wait()
}
