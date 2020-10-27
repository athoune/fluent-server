package server

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func New(eventHandler func(tag string, time time.Time, record map[string]interface{}) error) *Server {
	return &Server{
		eventHandler: eventHandler,
	}
}

type Server struct {
	eventHandler func(tag string, time time.Time, record map[string]interface{}) error
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
		fmt.Println("Hello", conn.RemoteAddr())
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

type Event struct {
	tag    string
	ts     time.Time
	record map[string]interface{}
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
		size, err := decoder.DecodeArrayLen()
		if err != nil {
			return err
		}
		events := make([]Event, size)
		for i := 0; i < size; i++ {
			c, err := decoder.PeekCode()
			if err != nil {
				return err
			}
			if !msgpcode.IsFixedArray(c) {
				return fmt.Errorf("Elem %d is not an array : %v", i, c)
			}
			l, err := decoder.DecodeArrayLen()
			if err != nil {
				return err
			}
			if l != 2 {
				return fmt.Errorf("Bad array length %v", l)
			}
			t, err := decoder.PeekCode()
			if err != nil {
				return err
			}
			var ts time.Time
			switch {
			case t == msgpcode.Uint32:
				tRaw, err := decoder.DecodeUint32()
				if err != nil {
					return err
				}
				ts = time.Unix(int64(tRaw), 0)
			case msgpcode.IsExt(t):
				fmt.Println("Ext")

			case msgpcode.IsFixedExt(t):
				fmt.Println("FixedExt")
			default:
				return fmt.Errorf("Unknown type %v", t)
			}
			record, err := decoder.DecodeMap()
			if err != nil {
				return err
			}
			events[i] = Event{tag, ts, record}
		}
		var option map[string]interface{}
		if l == 3 {
			option, err = decoder.DecodeMap()
			if err != nil {
				return err
			}
			fmt.Println("Option", option)
		}
		for _, event := range events {
			err = s.doEvent(event.tag, event.ts, event.record)
		}

	case msgpcode.IsBin(firstCode):
		fmt.Println("Bin")

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
			fmt.Println("option", option)
		}
		return s.doEvent(tag, time.Unix(int64(ts), 0), record)
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
func (s *Server) doEvent(tag string, ts time.Time, record map[string]interface{}) error {
	return s.eventHandler(tag, ts, record)
}
