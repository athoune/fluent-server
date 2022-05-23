package server

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"os"
	"sync"

	"github.com/athoune/fluent-server/message"
	"github.com/athoune/fluent-server/options"
)

// Server listening fluentd protocol
type Server struct {
	options    *options.FluentOptions
	useUDP     bool
	useMTLS    bool
	tlsConfig  *tls.Config
	listener   net.Listener
	waitListen *sync.WaitGroup
}

// New server, with an handler
func New(config *options.FluentOptions) (*Server, error) {
	if config.Logger == nil {
		config.Logger = log.Default()
	}
	var err error
	config.Hostname, err = os.Hostname()
	if err != nil {
		return nil, err
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	s := &Server{
		options:    config,
		waitListen: wg,
	}
	return s, nil
}

// New TLS server, with an handler
func NewTLS(config *options.FluentOptions, cfg *tls.Config) (*Server, error) {
	s, err := New(config)
	if err != nil {
		return nil, err
	}
	s.useUDP = false
	s.useMTLS = true
	s.tlsConfig = cfg
	return s, nil
}

// ListenAndServe an address
func (s *Server) ListenAndServe(address string) error {

	if s.useUDP {
		a, err := net.ResolveUDPAddr("udp", address)
		if err != nil {
			return err
		}
		listener, err := net.ListenUDP("udp", a)
		if err != nil {
			return err
		}
		go func() {
			defer listener.Close()
			buf := make([]byte, 1024)
			for {
				_, addr, err := listener.ReadFromUDP(buf)
				if err != nil {
					s.options.Logger.Printf("UDP read error : %v\n", err)
					continue
				}
				re, err := net.DialUDP("udp", nil, addr)
				if err != nil {
					s.options.Logger.Printf("UDP dial error : %v\n", err)
					continue
				}
				_, err = re.Write(buf)
				if err != nil {
					s.options.Logger.Printf("UDP write error : %v\n", err)
				}
				err = re.Close()
				if err != nil {
					s.options.Logger.Printf("UDP close error : %v\n", err)
				}
				s.options.Logger.Println("UDP Pong")
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
			session := message.NewSession(s.options, conn)
			err := session.Loop()
			if err != nil {
				if err == io.EOF {
					s.options.Logger.Println(conn.RemoteAddr(), "is closed")
				} else {
					s.options.Logger.Println("Error from", conn.RemoteAddr(), err)
				}
				return
			}
		}()
	}
}
