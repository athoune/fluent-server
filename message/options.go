package message

import (
	"github.com/athoune/fluent-server/msg"
	"github.com/vmihailenco/msgpack/v5"
)

func DecodeOption(decoder *msgpack.Decoder) (*msg.Option, error) {
	opt := &msg.Option{
		Stuff: make(map[string]interface{}),
	}

	option_l, err := decoder.DecodeMapLen()
	if err != nil {
		return nil, err
	}
	for i := 0; i < option_l; i++ {
		key, err := decoder.DecodeString()
		if err != nil {
			return nil, err
		}
		switch key {
		case "chunk":
			opt.Chunk, err = decoder.DecodeString()
		case "size":
			opt.Size, err = decoder.DecodeInt()
		case "compressed":
			opt.Compressed, err = decoder.DecodeString()
		default:
			opt.Stuff[key], err = decoder.DecodeInterface()
		}
		if err != nil {
			return nil, err
		}
	}
	return opt, nil
}
