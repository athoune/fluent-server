package server

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/factorysh/fluent-server/message"
)

func New(handler message.HandlerFunc) *Server {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	return &Server{
		reader:     message.New(handler),
		waitListen: wg,
	}
}

func NewTLS(handler message.HandlerFunc, cfg *tls.Config) *Server {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	return &Server{
		reader:     message.New(handler),
		useUDP:     false,
		useMTLS:    true,
		tlsConfig:  cfg,
		waitListen: wg,
	}
}

type Server struct {
	reader     *message.FluentReader
	useUDP     bool
	useMTLS    bool
	tlsConfig  *tls.Config
	listener   net.Listener
	waitListen *sync.WaitGroup
}

func (s *Server) ListenAndServe(address string) error {

	if s.useUDP {
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
	}
	var err error
	if s.useMTLS {
		s.listener, err = tls.Listen("tcp", address, s.tlsConfig)
	} else {
		s.listener, err = net.Listen("tcp", address)
	}
	if err != nil {
		return err
	}
	s.waitListen.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}
		log.Println("Connection from ", conn.RemoteAddr())
		go func() {
			session := &message.FluentSession{
				Reader: s.reader,
			}
			err := session.Loop(conn)
			if err != nil {
				if err == io.EOF {
					log.Println(conn.RemoteAddr(), "is closed")
				} else {
					log.Println("Error from", conn.RemoteAddr(), err)
				}
				return
			}
		}()
	}
}
