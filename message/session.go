package message

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/athoune/fluent-server/options"
	"github.com/athoune/fluent-server/wire"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

type Step int

const (
	WaitingForHelo Step = iota
	WaitingForPing
	WaitingForPong
	WaitingForEvents
)

type PasswordForKey func(string) string

type FluentSession struct {
	options        *options.FluentOptions
	nonce          []byte
	hashSalt       []byte
	Wire           *wire.Wire
	step           Step
	PasswordForKey PasswordForKey
	client         string
	messagesReader options.MessagesReader
}

func (s *FluentSession) debug(msg string, args ...any) {
	if s.options.Debug {
		s.options.Logger.Debug(msg, args...)
	}
}

func NewSession(opts *options.FluentOptions, conn io.ReadWriteCloser) *FluentSession {
	s := &FluentSession{
		Wire:    wire.New(conn),
		options: opts,
	}
	if s.options.SharedKey == "" {
		s.step = WaitingForEvents
	} else {
		s.step = WaitingForHelo
	}
	nconn, ok := conn.(net.Conn)
	if ok {
		s.client = nconn.RemoteAddr().String()
	}
	s.messagesReader = opts.MessagesReaderFactory(
		opts.Logger,
		opts.MessagesReaderConfig,
	)

	return s
}

func (s *FluentSession) Loop() error {
	for {
		err := s.handleMessage()
		if err != nil {
			if err == io.EOF {
				s.options.Logger.Info("connection closed", "client", s.client)
				return nil
			}
			s.options.Logger.Error("session error", "error", err, "client", s.client)
			return s.Wire.Close()
		}
	}
}

func (s *FluentSession) handleMessage() error {
	if s.step == WaitingForHelo {
		return s.DoHelo()
	}
	code, err := s.Wire.Decoder.PeekCode()
	if err != nil {
		return err
	}
	if code == msgpcode.Nil {
		return s.HandleHearthBeat()
	}
	if !msgpcode.IsFixedArray(code) {
		return fmt.Errorf("unexpected code %v", code)
	}
	l, err := s.Wire.Decoder.DecodeArrayLen()
	if err != nil {
		return err
	}
	if l == 0 {
		return errors.New("empty array")
	}
	_type, err := s.Wire.Decoder.DecodeString()
	if err != nil {
		return err
	}
	s.options.Logger.Debug("message type", "type", _type)
	switch s.step {
	case WaitingForPing:
		if _type != "PING" {
			return fmt.Errorf("waiting for a ping not %s", _type)
		}
		err = s.HandlePing(s.Wire, l, _type)
		if err != nil {
			return err
		}
		s.step = WaitingForEvents
	case WaitingForEvents:
		defer fmt.Println("Events")
		return s.HandleEvents(l, _type)
	default:
		return fmt.Errorf("unknown step : %v", s.step)
	}
	return nil
}

func (s *FluentSession) HandleHearthBeat() error {
	err := s.Wire.Decoder.DecodeNil()
	if err != nil {
		return err
	}
	s.options.Logger.Debug("heartbeat received")
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
