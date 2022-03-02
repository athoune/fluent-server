package message

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

func mockupServer(session *FluentSession) (net.Addr, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	go func() {
		err := func() error {
			conn, err := listener.Accept()
			if err != nil {
				return err
			}
			return session.Loop(conn)
		}()
		if err != nil {
			fmt.Println(err)
		}
	}()
	return listener.Addr(), nil
}

func TestSession(t *testing.T) {
	wg := &sync.WaitGroup{}
	var myRecord map[string]interface{}
	s, err := mockupServer(&FluentSession{
		Reader: New(func(tag string, time *time.Time, record map[string]interface{}) error {
			fmt.Println("record", record)
			wg.Done()
			myRecord = record
			return nil
		}),
	})
	assert.NoError(t, err)
	conn, err := net.Dial("tcp", s.String())
	assert.NoError(t, err)

	encoder := msgpack.NewEncoder(conn)
	decoder := msgpack.NewDecoder(conn)

	wg.Add(1)

	err = encoder.Encode([]interface{}{
		"tag.name",
		[]interface{}{
			[]interface{}{1441588984, map[string]interface{}{
				"message": "foo",
			}},
		},
		map[string]interface{}{
			"chunk": "p8n9gmxTQVC8/nh2wlKKeQ==",
			"size":  1,
		},
	})
	assert.NoError(t, err)

	wg.Wait()
	ack, err := decoder.DecodeMap()
	assert.NoError(t, err)
	assert.Equal(t, "p8n9gmxTQVC8/nh2wlKKeQ==", ack["ack"])
	assert.Equal(t, "foo", myRecord["message"])

}
