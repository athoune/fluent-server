package defaultreader

/*
Implement a default reader, using unmarshaled values, from mesgpack
*/

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/athoune/fluent-server/message"
	"github.com/athoune/fluent-server/options"
	"github.com/athoune/fluent-server/wire"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
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
			EventHandler: cfg["handler"].(HandlerFunc),
		}
	}
}

func (r *DefaultMessagesReader) MessageMode(wire *wire.Wire, tag string, l int) error {
	r.Logger.Println("Message Mode")
	if l > 4 {
		return fmt.Errorf("message too large: %d", l)
	}
	ts, err := message.DecodeTime(wire.Decoder)
	if err != nil {
		return err
	}
	record, err := wire.Decoder.DecodeMap()
	if err != nil {
		return err
	}
	if l == 4 {
		option, err := message.DecodeOption(wire.Decoder)
		if err != nil {
			return err
		}
		r.Logger.Println("option", option)
	}
	return r.EventHandler(tag, ts, record)
}

func (r *DefaultMessagesReader) PackedForwardMode(wire *wire.Wire, tag string, l int) error {
	firstCode, err := wire.Decoder.PeekCode()
	if err != nil {
		return err
	}
	var entries []byte
	switch {
	case msgpcode.IsBin(firstCode):
		entries, err = wire.Decoder.DecodeBytes()
		if err != nil {
			return err
		}
	case msgpcode.IsString(firstCode):
		return errors.New("PackedForward as string is deprecated")
	}
	var option *message.Option
	if l == 3 {
		option, err = message.DecodeOption(wire.Decoder)
		if err != nil {
			return err
		}
	}
	var _decoder *msgpack.Decoder
	if option != nil && option.Compressed == "gzip" {
		rr, err := gzip.NewReader(bytes.NewBuffer(entries))
		if err != nil {
			return err
		}
		r.Logger.Println("CompressedPackedForward")
		_decoder = msgpack.NewDecoder(rr)
	} else {
		_decoder = msgpack.NewDecoder(bytes.NewBuffer(entries))
	}
	if option != nil && option.Chunk != "" {
		err = message.Ack(wire, option.Chunk)
		if err != nil {
			return err
		}
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

func (r *DefaultMessagesReader) ForwardMode(wire *wire.Wire, tag string, l int) error {
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

	var option *message.Option
	if l == 3 { // there is options
		option, err := message.DecodeOption(wire.Decoder)
		if err != nil {
			return err
		}
		r.Logger.Println("options", option)
		if option.Chunk != "" {
			r.Logger.Println("ack", option.Chunk)
			err = message.Ack(wire, option.Chunk)
			if err != nil {
				return err
			}
		}
	}

	for _, event := range events {
		err = r.EventHandler(event.Tag, event.Ts, event.Record)
		if err != nil {
			return err
		}
	}
	// Server SHOULD close the connection silently with no response when the chunk option is not sent.
	if option == nil {
		return io.EOF
	} else if option.Chunk == "" {
		r.Logger.Println("No chunk, so I close the connection.")
		return io.EOF
	}

	return nil
}
