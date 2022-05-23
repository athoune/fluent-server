package message

import (
	"fmt"

	"github.com/athoune/fluent-server/msg"
	"github.com/athoune/fluent-server/wire"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func (s *FluentSession) decodeMessages(tag string, l int) error {
	firstCode, err := s.Wire.Decoder.PeekCode()
	if err != nil {
		return err
	}
	switch {
	case msgpcode.IsFixedArray(firstCode): // Forward mode
		err = s.messagesReader.ForwardMode(s.Wire, tag)
		if err != nil {
			return err
		}
		if l == 3 {
			_, err = handleChunk(s.Wire)
			if err != nil {
				return err
			}
		}
		s.debug("> message in forward mode")

	case msgpcode.IsString(firstCode): // PackedForward Mode
		return fmt.Errorf("PackedForward is old")
	case msgpcode.IsBin(firstCode): // PackedForward Mode
		blob, err := s.Wire.Decoder.DecodeBytes()
		if err != nil {
			return err
		}
		var opt *msg.Option
		if l == 3 {
			opt, err = handleChunk(s.Wire)
			if err != nil {
				return err
			}
		}
		err = s.messagesReader.PackedForwardMode(tag, blob, opt)
		s.debug("> message in packed forward mode")

	case firstCode == msgpcode.Uint32 || firstCode == msgpcode.Int32 || msgpcode.IsExt(firstCode): // Message Mode
		err = s.messagesReader.MessageMode(s.Wire, tag)
		s.debug("> message in message mode")
	default:
		err = fmt.Errorf("bad code %v", firstCode)
	}
	return err
}

func handleChunk(wire *wire.Wire) (*msg.Option, error) {
	opt, err := DecodeOption(wire.Decoder)
	if err != nil {
		return nil, err
	}
	if opt.Chunk != "" {
		err = Ack(wire, opt.Chunk)
		if err != nil {
			return nil, err
		}
	}
	return opt, nil
}
