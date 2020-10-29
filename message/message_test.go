package message

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

func TestHearthbeat(t *testing.T) {
	r, w := io.Pipe()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	f := New(func(tag string, time time.Time, record map[string]interface{}) error {
		assert.Equal(t, "beuha.aussi", tag)
		assert.Equal(t, int64(42), record["age"])
		wg.Done()
		return nil
	})

	go func() {
		err := f.Read(r)
		assert.NoError(t, err)
	}()

	go func() {
		b, err := msgpack.Marshal([]interface{}{
			"beuha.aussi",
			uint32(4807),
			map[string]interface{}{
				"name": "Bob",
				"age":  42,
			},
		})
		assert.NoError(t, err)
		w.Write(b)
	}()
	wg.Wait()
}
