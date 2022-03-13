package server

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	"github.com/factorysh/fluent-server/message"
)

func New(handler message.HandlerFunc) (*Server, error) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	s := &Server{
		reader:     message.New(handler),
		waitListen: wg,
		Logger:     log.Default(),
	}
	var err error
	s.Hostname, err = os.Hostname()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func NewTLS(handler message.HandlerFunc, cfg *tls.Config) (*Server, error) {
	s, err := New(handler)
	if err != nil {
		return nil, err
	}
	s.useUDP = false
	s.useMTLS = true
	s.tlsConfig = cfg
	s.Logger = log.Default()
	return s, nil
}

type Server struct {
	reader     *message.FluentReader
	useUDP     bool
	useMTLS    bool
	tlsConfig  *tls.Config
	listener   net.Listener
	waitListen *sync.WaitGroup
	SharedKey  string
	Hostname   string
	Logger     *log.Logger
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
				Reader:    s.reader,
				SharedKey: s.SharedKey,
				Hostname:  s.Hostname,
				Logger:    s.Logger,
			}
			err := session.Loop(conn)
			if err != nil {
				if err == io.EOF {
					s.Logger.Println(conn.RemoteAddr(), "is closed")
				} else {
					s.Logger.Println("Error from", conn.RemoteAddr(), err)
				}
				return
			}
		}()
	}
}
