package wire

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

func TestWire(t *testing.T) {
	conn, client := NewMockups()
	w := New(conn)
	err := client.Encoder.Encode(map[string]interface{}{
		"Hello": "World",
	})
	assert.NoError(t, err)
	err = w.Flush()
	assert.NoError(t, err)
	var r map[string]interface{}
	err = msgpack.Unmarshal(conn.In.Bytes(), &r)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"Hello": "World",
	}, r)
}
