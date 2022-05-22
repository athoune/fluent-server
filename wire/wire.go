package wire

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log"
	"net"

	"github.com/vmihailenco/msgpack/v5"
)

type Flusher interface {
	Flush() error
}

// Wire is the link with a client
type Wire struct {
	Encoder *msgpack.Encoder
	Decoder *msgpack.Decoder
	flusher Flusher
	closer  io.Closer
	Debug   func(string)
}

func New(conn io.ReadWriteCloser) *Wire {
	bufferedWriter := bufio.NewWriter(conn)
	return &Wire{
		flusher: bufferedWriter,
		Decoder: msgpack.NewDecoder(conn),
		Encoder: msgpack.NewEncoder(bufferedWriter),
		closer:  conn,
		//s.encoder.UseCompactInts(true)
		//s.encoder.UseCompactFloats(true)
		// FIXME : how can I set the logger?
		Debug: func(m string) {
			log.Println(m)
		},
	}
}

func (w *Wire) Close() error {
	return w.closer.Close()
}

func (w *Wire) Flush() error {
	return w.flusher.Flush()
}

type MockupClient struct {
	Encoder *msgpack.Encoder
	Decoder *msgpack.Decoder
}

func NewMockupBufferClient(a, b *bytes.Buffer) *MockupClient {
	return &MockupClient{
		Encoder: msgpack.NewEncoder(a),
		Decoder: msgpack.NewDecoder(b),
	}
}

type MockupServer struct {
	listen  net.Listener
	handler func(w *Wire) error
}

func (m *MockupServer) Start(ctx context.Context) {
	go func() {
		<-ctx.Done()
		m.listen.Close()
	}()
	go func() {
		for {
			conn, err := m.listen.Accept()
			if err != nil {
				break
			}
			go func() {
				w := New(conn)
				err = m.handler(w)
				if err != nil {
					log.Panic(err)
				}
			}()
		}
	}()
}

func NewMockups(handler func(w *Wire) error) (*MockupClient, *MockupServer, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, err
	}
	client, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		return nil, nil, err
	}

	return &MockupClient{
			Encoder: msgpack.NewEncoder(client),
			Decoder: msgpack.NewDecoder(client),
		},
		&MockupServer{
			listen:  l,
			handler: handler,
		},
		nil
}

type BufferCLoser struct {
	*bytes.Buffer
}

func (b *BufferCLoser) Close() error {
	return nil
}
