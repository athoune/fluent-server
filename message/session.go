package message

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

type Step int

const (
	WatingForHelo Step = iota
	WaitingForPing
	WaitingForPong
	WaitingForEvents
)

type PasswordForKey func(string) string

type FluentSession struct {
	nonce           []byte
	hashSalt        []byte
	encoder         *msgpack.Encoder
	decoder         *msgpack.Decoder
	Reader          *FluentReader
	SharedKey       string
	step            Step
	Hostname        string
	PasswordForKey  PasswordForKey
	shared_key_salt []byte
}

func (s *FluentSession) Loop(conn io.ReadWriteCloser) error {
	defer conn.Close()
	s.decoder = msgpack.NewDecoder(conn)
	s.encoder = msgpack.NewEncoder(conn)
	s.encoder.UseCompactInts(true)
	s.encoder.UseCompactFloats(true)

	if s.SharedKey == "" {
		s.step = WaitingForEvents
	} else {
		s.step = WatingForHelo
	}

	for {
		err := s.handleMessage()
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed", conn.(net.Conn).RemoteAddr().String())
				return nil
			}
			fmt.Println("Error : ", err)
			return err
		}
	}
}

func (s *FluentSession) handleMessage() error {
	if s.step == WatingForHelo {
		return s.doHelo()
	}
	code, err := s.decoder.PeekCode()
	fmt.Println("code", code)
	if err != nil {
		return err
	}
	if code == msgpcode.Nil {
		return s.HandleHearthBeat()
	}
	switch s.step {
	case WaitingForPing:
		return s.handlePing()
	case WaitingForEvents:
		return s.HandleEvents(code)
	default:
		return fmt.Errorf("unknown step : %v", s.step)
	}

}

func (s *FluentSession) HandleHearthBeat() error {
	err := s.decoder.DecodeNil()
	if err != nil {
		return err
	}
	fmt.Println("Hearthbeat")
	err = s.encoder.EncodeNil()
	if err != nil {
		return err
	}
	return nil
}

func (s *FluentSession) HandleEvents(code byte) error {
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
	if l < 2 {
		return errors.New("too short")
	}
	return s.decodeMessages(_type, l)
}

func (s *FluentSession) Ack(chunk string) error {
	return _map(s.encoder, "ack", chunk)
}
