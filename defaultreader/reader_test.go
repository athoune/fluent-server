package defaultreader

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/athoune/fluent-server/wire"
	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func TestReader(t *testing.T) {
	client, server := wire.NewMockups()
	defer client.Close()
	defer server.Close()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	reader := &DefaultMessagesReader{
		Logger: log.Default(),
		EventHandler: func(tag string, time *time.Time, record map[string]interface{}) error {
			wg.Done()
			return nil
		},
	}

	go func() {
		code, err := server.Decoder.PeekCode()
		assert.NoError(t, err)
		assert.True(t, msgpcode.IsFixedArray(code))
		l, err := server.Decoder.DecodeArrayLen()
		assert.NoError(t, err)
		assert.Equal(t, 2, l)
		err = reader.MessageMode(server, "myTag")
		assert.NoError(t, err)
	}()

	err := client.Encoder.Encode([]interface{}{1441588984, map[string]interface{}{
		"message": "foo",
	}})
	assert.NoError(t, err)
	err = client.Flush()
	assert.NoError(t, err)
	wg.Wait()
}
