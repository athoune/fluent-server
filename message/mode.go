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
		return fmt.Errorf("failed to peek message mode code: %w", err)
	}
	switch {
	case msgpcode.IsFixedArray(firstCode): // Forward mode
		err = s.messagesReader.ForwardMode(s.Wire, tag)
		if err != nil {
			return fmt.Errorf("failed to handle forward mode: %w", err)
		}
		if l == 3 {
			_, err = handleChunk(s.Wire)
			if err != nil {
				return fmt.Errorf("failed to handle chunk in forward mode: %w", err)
			}
		}
		s.debug("message in forward mode")

	case msgpcode.IsString(firstCode): // PackedForward Mode
		return fmt.Errorf("PackedForward is old")
	case msgpcode.IsBin(firstCode): // PackedForward Mode
		blob, err := s.Wire.Decoder.DecodeBytes()
		if err != nil {
			return fmt.Errorf("failed to decode packed forward blob: %w", err)
		}
		var opt *msg.Option
		if l == 3 {
			opt, err = handleChunk(s.Wire)
			if err != nil {
				return fmt.Errorf("failed to handle chunk in packed forward mode: %w", err)
			}
		}
		err = s.messagesReader.PackedForwardMode(tag, blob, opt)
		s.debug("message in packed forward mode")

	case firstCode == msgpcode.Uint32 || firstCode == msgpcode.Int32 || msgpcode.IsExt(firstCode): // Message Mode
		err = s.messagesReader.MessageMode(s.Wire, tag)
		s.debug("message in message mode")
	default:
		return fmt.Errorf("bad code %v", firstCode)
	}
	return err
}

func handleChunk(wire *wire.Wire) (*msg.Option, error) {
	opt, err := DecodeOption(wire.Decoder)
	if err != nil {
		return nil, fmt.Errorf("failed to decode option: %w", err)
	}
	if opt.Chunk != "" {
		err = Ack(wire, opt.Chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to send ack: %w", err)
		}
	}
	return opt, nil
}
