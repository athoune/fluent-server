package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"reflect"

	"github.com/vmihailenco/msgpack/v5"
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

func (s *Server) handler(conn net.Conn) {
	defer conn.Close()
	decoder := msgpack.NewDecoder(conn)
	var m []interface{}
	for {
		blob, err := decoder.DecodeInterface()
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Println("read error", err)
			return
		}
		if blob == nil {
			fmt.Println("Hearthbeat")
			continue
		}
		var ok bool
		m, ok = blob.([]interface{})
		if !ok {
			log.Println("Not an array", blob)
			return
		}
		if len(m) == 0 {
			log.Println("Empty array", blob)
			return
		}
		var type_ string
		type_, ok = m[0].(string)
		if !ok {
			log.Println("Type is not a string", m[0])
			return
		}
		switch type_ {
		case "HELO":
			s.doHelo()
		case "PING":
			s.doPing()
		case "PONG":
			s.doPong()
		default:
			if len(m) < 2 {
				log.Println("Too short", m)
				return
			}
			t := reflect.TypeOf(m[1])
			switch t.Kind() {
			case reflect.Array:
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
	}
}

func (s *Server) doHelo() {}
func (s *Server) doPing() {}
func (s *Server) doPong() {}
func (s *Server) doEvent(tag string, time uint32, record map[string]interface{}, option map[string]interface{}) {
	s.eventHandler(tag, time, record, option)
}
