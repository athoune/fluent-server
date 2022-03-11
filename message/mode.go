package message

import (
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func (s *FluentSession) decodeMessages(tag string, l int) error {
	firstCode, err := s.decoder.PeekCode()
	if err != nil {
		return err
	}
	switch {
	case msgpcode.IsFixedArray(firstCode): // Forward mode
		err = s.forwardMode(tag, l)

	case msgpcode.IsBin(firstCode) || msgpcode.IsString(firstCode): // PackedForward Mode
		err = s.packedForwardMode(tag, l)

	case firstCode == msgpcode.Uint32 || msgpcode.IsExt(firstCode): // Message Mode
		err = s.messageMode(tag, l)
	default:
		err = fmt.Errorf("bad code %v", firstCode)
	}
	return err
}

func (s *FluentSession) forwardMode(tag string, l int) error {
	log.Println("Forward mode")
	size, err := s.decoder.DecodeArrayLen()
	if err != nil {
		return err
	}
	events := make([]Event, size)
	for i := 0; i < size; i++ {
		ts, record, err := s.decodeEntry()
		if err != nil {
			return err
		}
		events[i] = Event{tag, ts, record}
	}

	var chunk string
	var compressed string

	if l == 3 { // there is options
		var key string
		var size int
		option_l, err := s.decoder.DecodeMapLen()
		if err != nil {
			return err
		}
		for i := 0; i < option_l; i++ {
			key, err = s.decoder.DecodeString()
			if err != nil {
				return err
			}
			switch key {
			case "chunk":
				chunk, err = s.decoder.DecodeString()
			case "size":
				size, err = s.decoder.DecodeInt()
			case "compressed":
				compressed, err = s.decoder.DecodeString()
			default:
				_, err = s.decoder.DecodeInterface()
			}
			if err != nil {
				return err
			}
		}
		fmt.Println("options", size, chunk, compressed)
		if chunk != "" {
			fmt.Println("ack", chunk)
			err = s.Ack(chunk)
			if err != nil {
				return err
			}
		}
	}
	for _, event := range events {
		err = s.Reader.eventHandler(event.tag, event.ts, event.record)
		if err != nil {
			return err
		}
	}
	//Server SHOULD close the connection silently with no response when the chunk option is not sent.
	if chunk == "" {
		return io.EOF
	}
	return nil
}

func (s *FluentSession) messageMode(tag string, l int) error {
	log.Println("Message Mode")
	if l > 4 {
		return fmt.Errorf("message too large: %d", l)
	}
	ts, err := decodeTime(s.decoder)
	if err != nil {
		return err
	}
	record, err := s.decoder.DecodeMap()
	if err != nil {
		return err
	}
	if l == 4 {
		option, err := s.decoder.DecodeMap()
		if err != nil {
			return err
		}
		fmt.Println("option", option)
	}
	return s.Reader.eventHandler(tag, ts, record)
}

func (s *FluentSession) packedForwardMode(tag string, l int) error {
	// TODO
	return errors.New("not implemented")
}
