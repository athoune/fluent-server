package server

import (
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func New(eventHandler func(tag string, time uint32, record map[string]interface{}, option map[string]interface{}) error) *Server {
	return &Server{
		eventHandler: eventHandler,
	}
}

type Server struct {
	eventHandler func(tag string, time uint32, record map[string]interface{}, option map[string]interface{}) error
}

func (s *Server) ListenAndServe(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go s.handler(conn)
	}
	return nil
}

func (s *Server) oneMessage(decoder *msgpack.Decoder) error {
	code, err := decoder.PeekCode()
	if err != nil {
		return err
	}
	if code == msgpcode.Nil {
		fmt.Println("Hearthbeat")
		return nil
	}
	if !msgpcode.IsFixedArray(code) {
		return errors.New("Not an array")
	}
	l, err := decoder.DecodeArrayLen()
	if err != nil {
		return err
	}
	if l == 0 {
		return errors.New("Empty array")
	}
	if l > 10 {
		return errors.New("Flood")
	}
	type_, err := decoder.DecodeString()
	if err != nil {
		return err
	}
	switch type_ {
	case "HELO":
		s.doHelo()
	case "PING":
		s.doPing()
	case "PONG":
		s.doPong()
	default:
		return s.doMessage(type_, decoder, l)
	}
	return nil

}

func (s *Server) doMessage(tag string, decoder *msgpack.Decoder, l int) error {
	if l < 2 {
		return errors.New("Too short")
	}
	firstCode, err := decoder.PeekCode()
	if err != nil {
		return err
	}
	switch {
	case msgpcode.IsFixedArray(firstCode):

	case msgpcode.IsBin(firstCode):

	case firstCode == msgpcode.Uint32:
		if l > 4 {
			return fmt.Errorf("Message too large: %d", l)
		}
		ts, err := decoder.DecodeUint32()
		if err != nil {
			return err
		}
		record, err := decoder.DecodeMap()
		if err != nil {
			return err
		}
		if l == 4 {
			option, err := decoder.DecodeMap()
			if err != nil {
				return err
			}
			return s.doEvent(tag, ts, record, option)
		}
		return s.doEvent(tag, ts, record, nil)
	default:
		return fmt.Errorf("Bad code %v", firstCode)
	}
	return nil
}

func (s *Server) handler(conn net.Conn) {
	defer conn.Close()
	decoder := msgpack.NewDecoder(conn)
	for {
		err := s.oneMessage(decoder)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func (s *Server) doHelo() {}
func (s *Server) doPing() {}
func (s *Server) doPong() {}
func (s *Server) doEvent(tag string, time uint32, record map[string]interface{}, option map[string]interface{}) error {
	return s.eventHandler(tag, time, record, option)
}
