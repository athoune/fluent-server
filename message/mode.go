package message

import (
	"fmt"

	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func (s *FluentSession) decodeMessages(tag string, l int) error {
	firstCode, err := s.Wire.Decoder.PeekCode()
	if err != nil {
		return err
	}
	switch {
	case msgpcode.IsFixedArray(firstCode): // Forward mode
		err = s.messagesReader.ForwardMode(s.Wire, tag, l)
		if err != nil {
			return err
		}
		if l == 3 {
			var opts map[string]interface{}
			err = s.Wire.Decoder.Decode(&opts)
			if err != nil {
				return err
			}
			chunk, ok := opts["chunk"]
			if ok {
				err = Ack(s.Wire, chunk.(string))
				if err != nil {
					return err
				}
			}
		}
		s.debug("> message in forward mode")

	case msgpcode.IsBin(firstCode) || msgpcode.IsString(firstCode): // PackedForward Mode
		err = s.messagesReader.PackedForwardMode(s.Wire, tag, l)
		s.debug("> message in packed forward mode")

	case firstCode == msgpcode.Uint32 || firstCode == msgpcode.Int32 || msgpcode.IsExt(firstCode): // Message Mode
		err = s.messagesReader.MessageMode(s.Wire, tag, l)
		s.debug("> message in message mode")
	default:
		err = fmt.Errorf("bad code %v", firstCode)
	}
	return err
}
