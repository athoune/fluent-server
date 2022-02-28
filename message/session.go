package message

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

type FluentSession struct {
	nonce    string
	hashSalt string
	pingStep bool
	encoder  *msgpack.Encoder
	decoder  *msgpack.Decoder
	Reader   *FluentReader
}

func (s *FluentSession) Loop(conn net.Conn) error {
	defer conn.Close()
	s.decoder = msgpack.NewDecoder(conn)
	s.encoder = msgpack.NewEncoder(conn)
	s.encoder.UseCompactInts(true)
	s.encoder.UseCompactFloats(true)
	for {
		err := s.handleMessage()
		if err != nil {
			return err
		}
	}
}

func (s *FluentSession) handleMessage() error {
	code, err := s.decoder.PeekCode()
	if err != nil {
		return err
	}
	if code == msgpcode.Nil {
		err = s.decoder.DecodeNil()
		if err != nil {
			return err
		}
		fmt.Println("Hearthbeat")
		return nil
	}
	if !msgpcode.IsFixedArray(code) {
		return errors.New("not an array")
	}
	l, err := s.decoder.DecodeArrayLen()
	if err != nil {
		return err
	}
	if l == 0 {
		return errors.New("empty array")
	}
	if l > 5 {
		return errors.New("flood")
	}
	_type, err := s.decoder.DecodeString()
	if err != nil {
		return err
	}
	switch _type {
	case "PING":
		return s.doPing()
	default: // It's a tag
		if l < 2 {
			return errors.New("too short")
		}
		return s.decodeMessages(_type, l)
	}
}

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

	case firstCode == msgpcode.Uint32: // Message Mode
		err = s.messageMode(tag, l)
	default:
		err = fmt.Errorf("bad code %v", firstCode)
	}
	return err
}

func (s *FluentSession) decodeEntry() (*time.Time, map[string]interface{}, error) {
	c, err := s.decoder.PeekCode()
	if err != nil {
		return nil, nil, err
	}
	if !msgpcode.IsFixedArray(c) {
		return nil, nil, fmt.Errorf("not an array : %v", c)
	}
	l, err := s.decoder.DecodeArrayLen()
	if err != nil {
		return nil, nil, err
	}
	if l != 2 {
		return nil, nil, fmt.Errorf("bad array length %v", l)
	}
	t, err := s.decoder.PeekCode()
	if err != nil {
		return nil, nil, err
	}
	var ts time.Time
	switch {
	case t == msgpcode.Uint32:
		tRaw, err := s.decoder.DecodeUint32()
		if err != nil {
			return nil, nil, err
		}
		ts = time.Unix(int64(tRaw), 0)
	case msgpcode.IsExt(t):
		id, len, err := s.decoder.DecodeExtHeader()
		if err != nil {
			return nil, nil, err
		}
		if id != 0 {
			return nil, nil, fmt.Errorf("unknown ext id %v", id)
		}
		if len != 8 {
			return nil, nil, fmt.Errorf("unknown ext id size %v", len)
		}
		b := make([]byte, len)
		l, err := s.decoder.Buffered().Read(b)
		if err != nil {
			return nil, nil, err
		}
		if l != len {
			return nil, nil, fmt.Errorf("read error, wrong size %v", l)
		}
		// https://pkg.go.dev/mod/github.com/vmihailenco/msgpack/v5@v5.0.0-rc.3#RegisterExt
		sec := binary.BigEndian.Uint32(b)
		usec := binary.BigEndian.Uint32(b[4:])
		ts = time.Unix(int64(sec), int64(usec))

	case msgpcode.IsFixedExt(t):
		fmt.Println("FixedExt")
	default:
		return nil, nil, fmt.Errorf("unknown type %v", t)
	}
	record, err := s.decoder.DecodeMap()
	if err != nil {
		return nil, nil, err
	}
	return &ts, record, nil
}

func (s *FluentSession) Ack(chunk string) error {
	err := s.encoder.EncodeMapLen(1)
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString("ack")
	if err != nil {
		return err
	}
	return s.encoder.EncodeString(chunk)
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
	if l == 3 { // there is options
		var chunk string
		var key string
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
			default:
				_, err = s.decoder.DecodeInterface()
			}
			if err != nil {
				return err
			}
		}
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
	return nil
}

func (s *FluentSession) messageMode(tag string, l int) error {
	log.Println("Message Mode")
	if l > 4 {
		return fmt.Errorf("message too large: %d", l)
	}
	ts, err := s.decoder.DecodeUint32()
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
	tz := time.Unix(int64(ts), 0)
	return s.Reader.eventHandler(tag, &tz, record)
}

func (s *FluentSession) packedForwardMode(tag string, l int) error {
	// TODO
	return nil
}
