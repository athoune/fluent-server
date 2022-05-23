package message

import (
	"encoding/binary"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

// See https://github.com/fluent/fluentd/wiki/Forward-Protocol-Specification-v1#eventtime-ext-format=
func TestTime(t *testing.T) {
	r, w := io.Pipe()
	encoder := msgpack.NewEncoder(w)
	decoder := msgpack.NewDecoder(r)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	now := time.Now()

	go func() {
		ts, err := DecodeTime(decoder)
		assert.NoError(t, err)
		assert.Equal(t, now.Round(time.Microsecond), *ts)
		wg.Done()
	}()

	encoder.EncodeExtHeader(0, 8)
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b, uint32(now.Unix()))
	binary.BigEndian.PutUint32(b[4:], uint32(now.Nanosecond()))
	_, err := encoder.Writer().Write(b)
	assert.NoError(t, err)
	wg.Wait()
}
