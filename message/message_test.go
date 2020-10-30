package message

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

type mockupConn struct {
	r io.Reader
	w io.Writer
}

func newMockup() (client, server *mockupConn) {
	rc, wc := io.Pipe()
	rs, ws := io.Pipe()
	return &mockupConn{rs, wc}, &mockupConn{rc, ws}
}

func (m *mockupConn) Read(p []byte) (n int, err error) {
	return m.r.Read(p)
}

func (m *mockupConn) Write(p []byte) (n int, err error) {
	return m.w.Write(p)
}

func TestHearthbeat(t *testing.T) {
	client, server := newMockup()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	f := New(func(tag string, time *time.Time, record map[string]interface{}) error {
		assert.Equal(t, "beuha.aussi", tag)
		assert.Equal(t, int64(42), record["age"])
		wg.Done()
		return nil
	})

	go func() {
		err := f.Listen(server)
		assert.NoError(t, err)
	}()

	go func() {
		b, err := msgpack.Marshal(nil)
		assert.NoError(t, err)
		client.Write(b)
		b, err = msgpack.Marshal([]interface{}{
			"beuha.aussi",
			uint32(4807),
			map[string]interface{}{
				"name": "Bob",
				"age":  42,
			},
		})
		assert.NoError(t, err)
		client.Write(b)
	}()
	wg.Wait()
}

func TestForwardMode(t *testing.T) {
	client, server := newMockup()
	wg := &sync.WaitGroup{}
	wg.Add(2)

	f := New(func(tag string, time *time.Time, record map[string]interface{}) error {
		assert.Equal(t, "beuha.aussi", tag)
		assert.Equal(t, int64(42), record["age"])
		wg.Done()
		return nil
	})

	go func() {
		err := f.Listen(server)
		assert.NoError(t, err)
	}()

	go func() {
		b, err := msgpack.Marshal([]interface{}{
			"beuha.aussi",
			[]interface{}{
				[]interface{}{
					uint32(4807),
					map[string]interface{}{
						"name": "Bob",
						"age":  42,
					},
				},
			},
			map[string]interface{}{
				"chunk": "oulalah",
			},
		})
		assert.NoError(t, err)
		client.Write(b)
		decoder := msgpack.NewDecoder(client)
		var r map[string]interface{}
		err = decoder.Decode(&r)
		assert.NoError(t, err)
		ack, ok := r["ack"]
		assert.True(t, ok)
		assert.Equal(t, "oulalah", ack)
		wg.Done()
	}()
	wg.Wait()
}
