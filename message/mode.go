package message

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"

	"github.com/vmihailenco/msgpack/v5"
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
		ts, record, err := decodeEntry(s.decoder)
		if err != nil {
			return err
		}
		events[i] = Event{tag, ts, record}
	}

	var option *Option

	if l == 3 { // there is options
		option, err := decodeOption(s.decoder)
		if err != nil {
			return err
		}
		fmt.Println("options", option)
		if option.Chunk != "" {
			fmt.Println("ack", option.Chunk)
			err = s.Ack(option.Chunk)
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
	if option != nil && option.Chunk == "" {
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
	firstCode, err := s.decoder.PeekCode()
	if err != nil {
		return err
	}
	var entries []byte
	switch {
	case msgpcode.IsBin(firstCode):
		entries, err = s.decoder.DecodeBytes()
		if err != nil {
			return err
		}
	case msgpcode.IsString(firstCode):
		return fmt.Errorf("PackedForward as string is deprecated")
	}
	var option *Option
	if l == 3 {
		option, err = decodeOption(s.decoder)
		if err != nil {
			return err
		}
	}
	var decoder *msgpack.Decoder
	if option != nil && option.Compressed == "gzip" {
		r, err := gzip.NewReader(bytes.NewBuffer(entries))
		if err != nil {
			return err
		}
		decoder = msgpack.NewDecoder(r)
	} else {
		decoder = msgpack.NewDecoder(bytes.NewBuffer(entries))
	}
	if option != nil && option.Chunk != "" {
		err = s.Ack(option.Chunk)
		if err != nil {
			return err
		}
	}
	for {
		ts, record, err := decodeEntry(decoder)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		err = s.Reader.eventHandler(tag, ts, record)
		if err != nil {
			return err
		}
	}
	return nil
}
