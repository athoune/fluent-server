package wire

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

type BufferCLoser struct {
	*bytes.Buffer
}

func (b *BufferCLoser) Close() error {
	return nil
}

func TestRead(t *testing.T) {
	b := &BufferCLoser{&bytes.Buffer{}}
	w := New(b)
	defer w.Close()
	encoder := msgpack.NewEncoder(b)
	err := encoder.Encode(map[string]interface{}{
		"Hello": "World",
	})
	assert.NoError(t, err)
	var v interface{}
	err = w.Decoder.Decode(&v)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"Hello": "World",
	}, v)
}

func TestWire(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var r map[string]interface{}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	client, server, err := NewMockups(func(w *Wire) error {
		err := w.Decoder.Decode(&r)
		if err != nil {
			return err
		}
		wg.Done()
		return nil
	})
	assert.NoError(t, err)
	server.Start(ctx)
	err = client.Encoder.Encode(map[string]interface{}{
		"Hello": "World",
	})
	assert.NoError(t, err)
	wg.Wait()
	assert.Equal(t, map[string]interface{}{
		"Hello": "World",
	}, r)

}
