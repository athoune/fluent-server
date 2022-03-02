package message

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func (s *FluentSession) decodeEntry() (*time.Time, map[string]interface{}, error) {
	c, err := s.decoder.PeekCode()
	if err != nil {
		return nil, nil, err
	}
	if !msgpcode.IsFixedArray(c) {
		return nil, nil, fmt.Errorf("not an array : %v", c)
	}
	l, err := s.decoder.DecodeArrayLen()
	if err != nil {
		return nil, nil, err
	}
	if l != 2 {
		return nil, nil, fmt.Errorf("bad array length %v", l)
	}
	t, err := s.decoder.PeekCode()
	if err != nil {
		return nil, nil, err
	}
	var ts time.Time
	switch {
	case t == msgpcode.Uint32:
		tRaw, err := s.decoder.DecodeUint32()
		if err != nil {
			return nil, nil, err
		}
		ts = time.Unix(int64(tRaw), 0)
	case msgpcode.IsExt(t):
		id, len, err := s.decoder.DecodeExtHeader()
		if err != nil {
			return nil, nil, err
		}
		if id != 0 {
			return nil, nil, fmt.Errorf("unknown ext id %v", id)
		}
		if len != 8 {
			return nil, nil, fmt.Errorf("unknown ext id size %v", len)
		}
		b := make([]byte, len)
		l, err := s.decoder.Buffered().Read(b)
		if err != nil {
			return nil, nil, err
		}
		if l != len {
			return nil, nil, fmt.Errorf("read error, wrong size %v", l)
		}
		// https://pkg.go.dev/mod/github.com/vmihailenco/msgpack/v5@v5.0.0-rc.3#RegisterExt
		sec := binary.BigEndian.Uint32(b)
		usec := binary.BigEndian.Uint32(b[4:])
		ts = time.Unix(int64(sec), int64(usec))

	case msgpcode.IsFixedExt(t):
		fmt.Println("FixedExt")
	default:
		return nil, nil, fmt.Errorf("unknown type %v", t)
	}
	record, err := s.decoder.DecodeMap()
	if err != nil {
		return nil, nil, err
	}
	return &ts, record, nil
}
