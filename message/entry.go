package message

import (
	"fmt"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func DecodeEntry(decoder *msgpack.Decoder) (*time.Time, map[string]interface{}, error) {
	c, err := decoder.PeekCode()
	if err != nil {
		return nil, nil, err
	}
	if !msgpcode.IsFixedArray(c) {
		return nil, nil, fmt.Errorf("not an array : %v", c)
	}
	l, err := decoder.DecodeArrayLen()
	if err != nil {
		return nil, nil, err
	}
	if l != 2 {
		return nil, nil, fmt.Errorf("bad array length %v", l)
	}
	ts, err := DecodeTime(decoder)
	if err != nil {
		return nil, nil, err
	}
	record, err := decoder.DecodeMap()
	if err != nil {
		return nil, nil, err
	}
	return ts, record, nil
}
