package message

import (
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
	ts, err := decodeTime(s.decoder)
	if err != nil {
		return nil, nil, err
	}
	record, err := s.decoder.DecodeMap()
	if err != nil {
		return nil, nil, err
	}
	return ts, record, nil
}
