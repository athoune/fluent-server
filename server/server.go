package server

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/vmihailenco/msgpack/v5"
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

		fmt.Println(m)
	}
}
