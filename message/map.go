package message

import "github.com/vmihailenco/msgpack/v5"

func _map(encoder *msgpack.Encoder, entries ...interface{}) error {
	l := len(entries)
	// TODO assert l is even
	encoder.EncodeMapLen(l / 2)
	for i := 0; i < l; i += 2 {
		err := encoder.EncodeString(entries[i].(string))
		if err != nil {
			return err
		}
		err = encoder.Encode(entries[i+1])
		if err != nil {
			return err
		}
	}
	return nil
}
