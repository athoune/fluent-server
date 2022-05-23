package wire

import (
	"io"
)

type PipeConn struct {
	reader io.Reader
	writer io.Writer
}

func (c *PipeConn) Write(p []byte) (int, error) {
	return c.writer.Write(p)
}

func (c *PipeConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (p *PipeConn) Close() error {
	return nil
}

func NewMockups() (*Wire, *Wire) {
	uplinkReader, downlinkWriter := io.Pipe()
	downlinkReader, uplinkWriter := io.Pipe()
	return New(&PipeConn{
			reader: uplinkReader,
			writer: uplinkWriter,
		}), New(&PipeConn{
			reader: downlinkReader,
			writer: downlinkWriter,
		})
}
