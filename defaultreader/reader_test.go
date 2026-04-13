package defaultreader

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
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
	done := make(chan struct{})

	handler := func(tag string, time *time.Time, record map[string]interface{}) error {
		wg.Done()
		return nil
	}

	factory := DefaultMessagesReaderFactory(handler)

	reader := factory(slog.Default(), nil)

	go func() {
		defer close(done)
		for {
			code, err := server.Decoder.PeekCode()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				t.Errorf("peek code error: %v", err)
				return
			}
			if !msgpcode.IsFixedArray(code) {
				t.Errorf("expected fixed array, got %v", code)
				return
			}
			l, err := server.Decoder.DecodeArrayLen()
			if err != nil {
				t.Errorf("decode array length error: %v", err)
				return
			}
			firstCode, err := server.Decoder.PeekCode()
			if err != nil {
				t.Errorf("peek first code error: %v", err)
				return
			}
			switch {
			case firstCode == msgpcode.Uint32 || firstCode == msgpcode.Int32 || msgpcode.IsExt(firstCode): // Message Mode
				if l != 2 {
					t.Errorf("expected array length 2, got %d", l)
					return
				}
				err = reader.MessageMode(server, "myTag")
				if err != nil {
					t.Errorf("message mode error: %v", err)
					return
				}
			case msgpcode.IsFixedArray(firstCode): // Forward mode
				err = reader.ForwardMode(server, "myTag")
				if err != nil {
					t.Errorf("forward mode error: %v", err)
					return
				}
			case msgpcode.IsBin(firstCode): // PackedForward Mode
				blob, err := server.Decoder.DecodeBytes()
				if err != nil {
					t.Errorf("decode bytes error: %v", err)
					return
				}
				err = reader.PackedForwardMode("myTag", blob, &msg.Option{})
				if err != nil {
					t.Errorf("packed forward mode error: %v", err)
					return
				}
			default:
				t.Errorf("unexpected code: %v", firstCode)
				return
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

	// Close client to trigger EOF in the goroutine and wait for it to finish
	client.Close()
	<-done
}
