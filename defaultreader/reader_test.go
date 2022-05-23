package defaultreader

import (
	"bytes"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/athoune/fluent-server/msg"
	"github.com/athoune/fluent-server/wire"
	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func TestReader(t *testing.T) {
	var err error
	client, server := wire.NewMockups()
	defer client.Close()
	defer server.Close()

	wg := &sync.WaitGroup{}

	handler := func(tag string, time *time.Time, record map[string]interface{}) error {
		wg.Done()
		return nil
	}

	factory := DefaultMessagesReaderFactory(handler)

	reader := factory(log.Default(), nil)

	go func() {
		for {
			code, err := server.Decoder.PeekCode()
			assert.NoError(t, err)
			assert.True(t, msgpcode.IsFixedArray(code))
			l, err := server.Decoder.DecodeArrayLen()
			assert.NoError(t, err)
			firstCode, err := server.Decoder.PeekCode()
			assert.NoError(t, err)
			switch {
			case firstCode == msgpcode.Uint32 || firstCode == msgpcode.Int32 || msgpcode.IsExt(firstCode): // Message Mode
				assert.Equal(t, 2, l)
				err = reader.MessageMode(server, "myTag")
				assert.NoError(t, err)
			case msgpcode.IsFixedArray(firstCode): // Forward mode
				err = reader.ForwardMode(server, "myTag")
				assert.NoError(t, err)
			case msgpcode.IsBin(firstCode): // PackedForward Mode
				blob, err := server.Decoder.DecodeBytes()
				assert.NoError(t, err)
				err = reader.PackedForwardMode("myTag", blob, &msg.Option{})
				assert.NoError(t, err)
			default:
				assert.True(t, false)
			}
		}
	}()

	wg.Add(1)
	err = client.Encoder.Encode([]interface{}{1441588984, map[string]interface{}{
		"message": "foo",
	}})
	assert.NoError(t, err)
	err = client.Flush()
	assert.NoError(t, err)
	wg.Wait()

	wg.Add(2)
	err = client.Encoder.Encode([]interface{}{
		[]interface{}{
			[]interface{}{1441588984, map[string]interface{}{
				"message": "foo",
			}},
			[]interface{}{1441588985, map[string]interface{}{
				"message": "bar",
			}},
		},
	})
	assert.NoError(t, err)
	err = client.Flush()
	assert.NoError(t, err)
	wg.Wait()

	wg.Add(2)
	buff := &bytes.Buffer{}
	encoder := msgpack.NewEncoder(buff)
	err = encoder.Encode([]interface{}{1441588984, map[string]interface{}{
		"message": "foo",
	}})
	assert.NoError(t, err)
	err = encoder.Encode([]interface{}{1441588985, map[string]interface{}{
		"message": "bar",
	}})
	assert.NoError(t, err)
	err = client.Encoder.Encode([]interface{}{
		buff.Bytes(),
	})
	assert.NoError(t, err)
	err = client.Flush()
	assert.NoError(t, err)
	wg.Wait()

}
