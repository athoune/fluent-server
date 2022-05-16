package defaultreader

import (
	"bytes"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/athoune/fluent-server/wire"
	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func TestReader(t *testing.T) {
	b := &wire.BufferCLoser{&bytes.Buffer{}}
	w := wire.New(b)
	defer w.Close()
	encoder := msgpack.NewEncoder(b)
	wg := &sync.WaitGroup{}
	reader := &DefaultMessagesReader{
		Logger: log.Default(),
		EventHandler: func(tag string, time *time.Time, record map[string]interface{}) error {
			wg.Done()
			return nil
		},
	}

	wg.Add(1)
	err := encoder.Encode([]interface{}{1441588984, map[string]interface{}{
		"message": "foo",
	}})
	assert.NoError(t, err)
	code, err := w.Decoder.PeekCode()
	assert.NoError(t, err)
	assert.True(t, msgpcode.IsFixedArray(code))
	l, err := w.Decoder.DecodeArrayLen()
	assert.NoError(t, err)
	assert.Equal(t, 2, l)
	err = reader.MessageMode(w, "myTag", l)
	assert.NoError(t, err)
	wg.Wait()
}
