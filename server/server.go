package server

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/factorysh/fluent-server/message"
	"github.com/tinylib/msgp/msgp"
)

func New() *Server {
	return &Server{}
}

type Server struct {
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
	buf := make([]byte, 2048)
	var m message.Message
	for {
		_, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			log.Println("read error", err)
			return
		}
		if msgp.IsNil(buf) {
			buf = buf[1:]
			continue
		}
		buf, err = m.UnmarshalMsg(buf)
		if err != nil {
			_, ok := err.(msgp.TypeError)
			if !ok {
				log.Println("read error", err)
				return
			}
			var size message.Size
			buf, err = size.UnmarshalMsg(buf)
			if err != nil {
				log.Println("read error", err)
				return
			}
			fmt.Println(size)
		} else {
			fmt.Println(m)
		}
	}
}
