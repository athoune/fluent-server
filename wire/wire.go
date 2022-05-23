package wire

import (
	"bufio"
	"io"
	"log"

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
