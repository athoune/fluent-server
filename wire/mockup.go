package wire

import (
	"io"
)

type PipeConn struct {
	reader      io.Reader
	writer      io.Writer
	writeCloser io.Closer
}

func (c *PipeConn) Write(p []byte) (int, error) {
	return c.writer.Write(p)
}

func (c *PipeConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c *PipeConn) Close() error {
	if c.writeCloser != nil {
		return c.writeCloser.Close()
	}
	return nil
}

func NewMockups() (*Wire, *Wire) {
	uplinkReader, downlinkWriter := io.Pipe()
	downlinkReader, uplinkWriter := io.Pipe()
	return New(&PipeConn{
			reader:      uplinkReader,
			writer:      uplinkWriter,
			writeCloser: uplinkWriter,
		}), New(&PipeConn{
			reader:      downlinkReader,
			writer:      downlinkWriter,
			writeCloser: downlinkWriter,
		})
}
