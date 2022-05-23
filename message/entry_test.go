package message

import (
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

func TestEntry(t *testing.T) {
	r, w := io.Pipe()
	encoder := msgpack.NewEncoder(w)
	decoder := msgpack.NewDecoder(r)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		ts, data, err := DecodeEntry(decoder)
		assert.NoError(t, err)
		assert.Equal(t, map[string]interface{}{
			"message": "foo",
		}, data)
		assert.Equal(t, int64(1441588984), ts.Unix())
		wg.Done()
	}()

	err := encoder.Encode([]interface{}{1441588984, map[string]interface{}{
		"message": "foo",
	}})
	assert.NoError(t, err)
	wg.Wait()
}
