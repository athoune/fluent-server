package message

import (
	"errors"
	"fmt"
	"io"

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

func (s *FluentSession) Loop(conn io.ReadWriteCloser) error {
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

func (s *FluentSession) Ack(chunk string) error {
	return _map(s.encoder, "ack", chunk)
}
