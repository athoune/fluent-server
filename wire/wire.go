package wire

import (
	"bufio"
	"bytes"
	"io"

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
	}
}

func (w *Wire) Close() error {
	return w.closer.Close()
}

func (w *Wire) Flush() error {
	return w.flusher.Flush()
}

type MockupConn struct {
	In  *bytes.Buffer
	Out *bytes.Buffer
}

type MockupClient struct {
	Encoder *msgpack.Encoder
	Decoder *msgpack.Decoder
}

func NewMockups() (*MockupConn, *MockupClient) {
	conn := &MockupConn{
		In:  &bytes.Buffer{},
		Out: &bytes.Buffer{},
	}

	return conn, &MockupClient{
		Encoder: msgpack.NewEncoder(conn.In),
		Decoder: msgpack.NewDecoder(conn.Out),
	}
}

func (m *MockupConn) Write(p []byte) (n int, err error) {
	return m.In.Write(p)
}

func (m *MockupConn) Read(p []byte) (n int, err error) {
	return m.Out.Read(p)
}

func (m *MockupConn) Close() error {
	return nil
}
