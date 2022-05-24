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
	udpConn    *net.UDPConn
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
		s.udpConn, err = net.ListenUDP("udp", a)
		if err != nil {
			return err
		}
		defer s.udpConn.Close()
		s.options.Logger.Printf("Listening UDP %s => %s", s.udpConn.LocalAddr(), s.udpConn.RemoteAddr())
		go func() {
			buf := make([]byte, 1024)
			for {
				n, remoteAddr, err := s.udpConn.ReadFromUDP(buf)
				if err != nil {
					s.options.Logger.Printf("UDP read error : %v\n", err)
					continue
				}
				_, err = s.udpConn.WriteToUDP(buf[:n], remoteAddr)
				if err != nil {
					s.options.Logger.Printf("UDP write error : %v\n", err)
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
