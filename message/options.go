package message

import "github.com/vmihailenco/msgpack/v5"

type Option struct {
	Size       int
	Chunk      string
	Compressed string
	Stuff      map[string]interface{}
}

func decodeOption(decoder *msgpack.Decoder) (*Option, error) {
	opt := &Option{
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
