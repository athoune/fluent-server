package server

import (
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func New(eventHandler func(tag string, time uint32, record map[string]interface{}, option map[string]interface{})) *Server {
	return &Server{
		eventHandler: eventHandler,
	}
}

type Server struct {
	eventHandler func(tag string, time uint32, record map[string]interface{}, option map[string]interface{})
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
	fmt.Println("code", code)
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
	fmt.Println("len", l)
	if l == 0 {
		return errors.New("Empty array")
	}
	if l > 10 {
		return errors.New("Flood")
	}
	type_, err := decoder.DecodeString()
	if err != nil {
		fmt.Println("Paf", err)
		return err
	}
	fmt.Println("Type:", type_)
	switch type_ {
	case "HELO":
		s.doHelo()
	case "PING":
		s.doPing()
	case "PONG":
		s.doPong()
	default:
		return s.doMessage(decoder, l)
	}
	return nil

}

func (s *Server) doMessage(decoder *msgpack.Decoder, l int) error {
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
		ts, err := decoder.DecodeUint32()
		if err != nil {
			return err
		}
		fmt.Println("ts", ts)
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
		/*
				if len(m) < 2 {
					log.Println("Too short", m)
					return
				}
				if codes.IsFixedArray(m[1][0]) {
					fmt.Println("a batch of events")
					for _, blob := range m[1].([]interface{}) {
						var evt []interface{}
						evt, ok = blob.([]interface{})
						if !ok {
							fmt.Println("bad event format:", blob)
							return
						}
						if len(evt) != 2 {
							fmt.Println("bad event size:", evt)
							return
						}
						var time uint32
						time, ok = evt[0].(uint32)
						if !ok {
							fmt.Println("Bad time format:", evt[0])
							return
						}
						var record map[string]interface{}
						record, ok := evt[1].(map[string]interface{})
						if !ok {
							fmt.Println("Bad record format:", evt[1])
							return
						}
						s.doEvent(type_, time, record, nil)
					}
				case reflect.Uint32:
					var record map[string]interface{}
					record, ok = m[2].(map[string]interface{})
					if !ok {
						fmt.Println("Bad record type:", m[2])
						return
					}
					if len(m) == 3 {
						s.doEvent(type_, m[1].(uint32), record, nil)
					} else {
						var option map[string]interface{}
						option, ok = m[3].(map[string]interface{})
						if !ok {
							fmt.Println("Bad option type:", m[3])
							return
						}
						s.doEvent(type_, m[1].(uint32), record, option)
					}
				}
			}
		*/
	}
}

func (s *Server) doHelo() {}
func (s *Server) doPing() {}
func (s *Server) doPong() {}
func (s *Server) doEvent(tag string, time uint32, record map[string]interface{}, option map[string]interface{}) {
	s.eventHandler(tag, time, record, option)
}
