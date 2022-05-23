package wire

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWire(t *testing.T) {
	client, server := NewMockups()
	defer client.Close()
	defer server.Close()

	var r map[string]interface{}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		err := server.Decoder.Decode(&r)
		assert.NoError(t, err)
		wg.Done()
	}()

	err := client.Encoder.Encode(map[string]interface{}{
		"Hello": "World",
	})
	assert.NoError(t, err)
	client.Flush()
	wg.Wait()
	assert.Equal(t, map[string]interface{}{
		"Hello": "World",
	}, r)

}
