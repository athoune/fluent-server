package message

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
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
	flusher        Flusher
	Logger         *log.Logger
	Users          func(string) []byte
	Debug          bool
}

type Flusher interface {
	Flush() error
}

func (s *FluentSession) Flush() error {
	return s.flusher.Flush()
}

func (s *FluentSession) debug(v ...interface{}) {
	if s.Debug {
		log.Println("🐞", fmt.Sprint(v...))
	}
}

func (s *FluentSession) Loop(conn io.ReadWriteCloser) error {
	defer conn.Close()
	bufferedWriter := bufio.NewWriter(conn)
	s.flusher = bufferedWriter
	s.decoder = msgpack.NewDecoder(conn)
	s.encoder = msgpack.NewEncoder(bufferedWriter)
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
			var client string
			nconn, ok := conn.(net.Conn)
			if ok {
				client = nconn.RemoteAddr().String()
			} else {
				client = ""
			}
			if err == io.EOF {
				s.Logger.Println("Connection closed", client)
				return nil
			}
			s.Logger.Println("Error : ", err, client)
			return conn.Close()
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
	if !msgpcode.IsFixedArray(code) {
		return fmt.Errorf("unexpected code %v", code)
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
	s.Logger.Printf("Type : [%s]\n", _type)
	switch s.step {
	case WaitingForPing:
		if _type != "PING" {
			return fmt.Errorf("waiting for a ping not %s", _type)
		}
		return s.handlePing(l, _type)
	case WaitingForEvents:
		defer fmt.Println("Events")
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
	s.Logger.Println("Hearthbeat")
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
