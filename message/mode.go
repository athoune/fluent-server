package message

import (
	"fmt"

	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func (s *FluentSession) decodeMessages(tag string, l int) error {
	firstCode, err := s.decoder.PeekCode()
	if err != nil {
		return err
	}
	switch {
	case msgpcode.IsFixedArray(firstCode): // Forward mode
		err = s.MessagesReader.ForwardMode(s.decoder, tag, l)
		s.debug("> message in forward mode")

	case msgpcode.IsBin(firstCode) || msgpcode.IsString(firstCode): // PackedForward Mode
		err = s.MessagesReader.PackedForwardMode(s.decoder, tag, l)
		s.debug("> message in packed forward mode")

	case firstCode == msgpcode.Uint32 || firstCode == msgpcode.Int32 || msgpcode.IsExt(firstCode): // Message Mode
		err = s.MessagesReader.MessageMode(s.decoder, tag, l)
		s.debug("> message in message mode")
	default:
		err = fmt.Errorf("bad code %v", firstCode)
	}
	return err
}
