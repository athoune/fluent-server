package message

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func DecodeTime(decoder *msgpack.Decoder) (*time.Time, error) {
	t, err := decoder.PeekCode()
	if err != nil {
		return nil, fmt.Errorf("failed to peek time code: %w", err)
	}
	var ts time.Time
	switch {
	case t == msgpcode.Uint32:
		tRaw, err := decoder.DecodeUint32()
		if err != nil {
			return nil, fmt.Errorf("failed to decode uint32 timestamp: %w", err)
		}
		ts = time.Unix(int64(tRaw), 0)
	case t == msgpcode.Int32:
		tRaw, err := decoder.DecodeInt32()
		if err != nil {
			return nil, fmt.Errorf("failed to decode int32 timestamp: %w", err)
		}
		ts = time.Unix(int64(tRaw), 0)
	case msgpcode.IsExt(t):
		id, len, err := decoder.DecodeExtHeader()
		if err != nil {
			return nil, fmt.Errorf("failed to decode ext header: %w", err)
		}
		if id != 0 {
			return nil, fmt.Errorf("unknown ext id %v", id)
		}
		if len != 8 {
			return nil, fmt.Errorf("unknown ext id size %v", len)
		}
		b := make([]byte, len)
		l, err := decoder.Buffered().Read(b)
		if err != nil {
			return nil, fmt.Errorf("failed to read ext data: %w", err)
		}
		if l != len {
			return nil, fmt.Errorf("read error, wrong size %v", l)
		}
		// https://pkg.go.dev/mod/github.com/vmihailenco/msgpack/v5@v5.0.0-rc.3#RegisterExt
		sec := binary.BigEndian.Uint32(b)
		usec := binary.BigEndian.Uint32(b[4:])
		ts = time.Unix(int64(sec), int64(usec))
	case msgpcode.IsFixedExt(t):
		return nil, fmt.Errorf("FixedExt")
	default:
		return nil, fmt.Errorf("unknown type %v", t)
	}
	return &ts, nil
}
