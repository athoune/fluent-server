package defaultreader

/*
Implement a default reader, using unmarshaled values, from mesgpack
*/

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"time"

	"github.com/athoune/fluent-server/message"
	"github.com/athoune/fluent-server/msg"
	"github.com/athoune/fluent-server/options"
	"github.com/athoune/fluent-server/wire"
	"github.com/vmihailenco/msgpack/v5"
)

// HandlerFunc handles an event
type HandlerFunc func(tag string, time *time.Time, record map[string]interface{}) error

type DefaultMessagesReader struct {
	Logger       *log.Logger
	EventHandler HandlerFunc
}

func DefaultMessagesReaderFactory(eventHandler HandlerFunc) options.MessagesReaderFactory {
	return func(logger *log.Logger, cfg map[string]interface{}) options.MessagesReader {
		return &DefaultMessagesReader{
			Logger:       logger,
			EventHandler: eventHandler,
		}
	}
}

func (r *DefaultMessagesReader) MessageMode(wire *wire.Wire, tag string) error {
	r.Logger.Println("Message Mode")
	ts, err := message.DecodeTime(wire.Decoder)
	if err != nil {
		return err
	}
	record, err := wire.Decoder.DecodeMap()
	if err != nil {
		return err
	}
	return r.EventHandler(tag, ts, record)
}

func (r *DefaultMessagesReader) PackedForwardMode(tag string, entries []byte, opt *msg.Option) error {
	var _decoder *msgpack.Decoder
	if opt != nil && opt.Compressed == "gzip" {
		rr, err := gzip.NewReader(bytes.NewBuffer(entries))
		if err != nil {
			return err
		}
		r.Logger.Println("CompressedPackedForward")
		_decoder = msgpack.NewDecoder(rr)
	} else {
		_decoder = msgpack.NewDecoder(bytes.NewBuffer(entries))
	}
	for {
		ts, record, err := message.DecodeEntry(_decoder)
		if err != nil {
			if err == io.EOF { // the PackedForward is ended, it's ok.
				return nil
			}
			return err
		}
		err = r.EventHandler(tag, ts, record)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *DefaultMessagesReader) ForwardMode(wire *wire.Wire, tag string) error {
	r.Logger.Println("Forward mode")
	size, err := wire.Decoder.DecodeArrayLen()
	if err != nil {
		return err
	}
	events := make([]message.Event, size)
	for i := 0; i < size; i++ {
		ts, record, err := message.DecodeEntry(wire.Decoder)
		if err != nil {
			return err
		}
		events[i] = message.Event{
			Tag:    tag,
			Ts:     ts,
			Record: record,
		}
	}

	for _, event := range events {
		err = r.EventHandler(event.Tag, event.Ts, event.Record)
		if err != nil {
			return err
		}
	}
	// Server SHOULD close the connection silently with no response when the chunk option is not sent.
	/*
		if option == nil {
			return io.EOF
		} else if option.Chunk == "" {
			r.Logger.Println("No chunk, so I close the connection.")
			return io.EOF
		}
	*/

	return nil
}
