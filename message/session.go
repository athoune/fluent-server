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
	nonce          []byte
	hashSalt       []byte
	encoder        *msgpack.Encoder
	decoder        *msgpack.Decoder
	Reader         *FluentReader
	SharedKey      string
	step           Step
	Hostname       string
	PasswordForKey PasswordForKey
}

func (s *FluentSession) Loop(conn io.ReadWriteCloser) error {
	defer conn.Close()
	s.decoder = msgpack.NewDecoder(conn)
	s.encoder = msgpack.NewEncoder(conn)
	//s.encoder.UseCompactInts(true)
	//s.encoder.UseCompactFloats(true)

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
	if err != nil {
		return err
	}
	if code == msgpcode.Nil {
		return s.HandleHearthBeat()
	}
	l, err := s.decoder.DecodeArrayLen()
	if err != nil {
		return err
	}
	if l == 0 {
		return errors.New("empty array")
	}
	_type, err := s.decoder.DecodeString()
	if err != nil {
		return err
	}
	fmt.Printf("Type : [%s]\n", _type)
	switch s.step {
	case WaitingForPing:
		if _type != "PING" {
			return fmt.Errorf("waiting for a ping not %s", _type)
		}
		return s.handlePing(l, _type)
	case WaitingForEvents:
		return s.HandleEvents(l, _type)
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
	/*
		err = s.encoder.EncodeNil()
		if err != nil {
			return err
		}
	*/
	return nil
}

func (s *FluentSession) HandleEvents(l int, _type string) error {
	if l > 5 {
		return errors.New("flood")
	}
	if l < 2 {
		return errors.New("too short")
	}
	return s.decodeMessages(_type, l)
}
