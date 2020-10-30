package server

import (
	"fmt"
	"log"
	"net"

	"github.com/factorysh/fluent-server/message"
)

func New(handler message.HandlerFunc) *Server {
	return &Server{
		reader: message.New(handler),
	}
}

type Server struct {
	reader *message.FluentReader
}

func (s *Server) ListenAndServe(address string) error {
	go func() {
		a, err := net.ResolveUDPAddr("udp", address)
		if err != nil {
			panic(err)
		}
		listener, err := net.ListenUDP("udp", a)
		if err != nil {
			panic(err)
		}
		defer listener.Close()
		buf := make([]byte, 1024)
		for {
			_, addr, err := listener.ReadFromUDP(buf)
			if err != nil {
				fmt.Println(err)
				return
			}
			re, _ := net.DialUDP("udp", nil, addr)
			defer re.Close()
			re.Write(buf)
			fmt.Println("Pong")
		}
	}()
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
		go func() {
			defer conn.Close()
			err := s.reader.Listen(conn)
			if err != nil {
				log.Println(err)
				return
			}
		}()
	}
}
