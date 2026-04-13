package server

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"

	"github.com/athoune/fluent-server/message"
	"github.com/athoune/fluent-server/options"
)

// Server listening fluentd protocol
type Server struct {
	options     *options.FluentOptions
	useUDP      bool
	useMTLS     bool
	tlsConfig   *tls.Config
	listener    net.Listener
	udpConn     *net.UDPConn
	waitListen  *sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	connections sync.WaitGroup
	mu          sync.Mutex
	closed      bool
}

// New server, with an handler
func New(config *options.FluentOptions) (*Server, error) {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	var err error
	config.Hostname, err = os.Hostname()
	if err != nil {
		return nil, err
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		options:    config,
		waitListen: wg,
		ctx:        ctx,
		cancel:     cancel,
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
		s.options.Logger.Info("listening UDP", "local", s.udpConn.LocalAddr(), "remote", s.udpConn.RemoteAddr())
		go func() {
			buf := make([]byte, 1024)
			for {
				select {
				case <-s.ctx.Done():
					return
				default:
				}
				n, remoteAddr, err := s.udpConn.ReadFromUDP(buf)
				if err != nil {
					s.options.Logger.Error("UDP read error", "error", err)
					continue
				}
				_, err = s.udpConn.WriteToUDP(buf[:n], remoteAddr)
				if err != nil {
					s.options.Logger.Error("UDP write error", "error", err)
				}
				s.options.Logger.Debug("UDP Pong")
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
		select {
		case <-s.ctx.Done():
			return nil
		default:
		}
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		s.connections.Add(1)
		s.options.Logger.Info("new connection", "remote", conn.RemoteAddr())
		go func() {
			defer s.connections.Done()
			session := message.NewSession(s.options, conn)
			err := session.Loop()
			if err != nil {
				if errors.Is(err, io.EOF) {
					s.options.Logger.Info("connection closed", "remote", conn.RemoteAddr())
				} else {
					s.options.Logger.Error("connection error", "remote", conn.RemoteAddr(), "error", err)
				}
				return
			}
		}()
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	s.options.Logger.Info("shutting down server")
	s.cancel()

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			return err
		}
	}

	if s.udpConn != nil {
		s.udpConn.Close()
	}

	s.connections.Wait()
	s.options.Logger.Info("server shutdown complete")
	return nil
}
