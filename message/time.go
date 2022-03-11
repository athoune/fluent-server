package message

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func decodeTime(decoder *msgpack.Decoder) (*time.Time, error) {
	t, err := decoder.PeekCode()
	if err != nil {
		return nil, err
	}
	var ts time.Time
	switch {
	case t == msgpcode.Uint32:
		tRaw, err := decoder.DecodeUint32()
		if err != nil {
			return nil, err
		}
		ts = time.Unix(int64(tRaw), 0)
	case msgpcode.IsExt(t):
		id, len, err := decoder.DecodeExtHeader()
		if err != nil {
			return nil, err
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
			return nil, err
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
