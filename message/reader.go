package message

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

type DefaultMessagesReader struct {
	Logger *log.Logger
	Reader *FluentReader
	Ack    func(string) error
}

func UseDefaultMessageReader(session *FluentSession) {
	session.MessagesReader = &DefaultMessagesReader{
		Logger: session.Logger,
		Reader: session.Reader,
		Ack:    session.Ack,
	}
}

func (r *DefaultMessagesReader) MessageMode(decoder *msgpack.Decoder, tag string, l int) error {
	r.Logger.Println("Message Mode")
	if l > 4 {
		return fmt.Errorf("message too large: %d", l)
	}
	ts, err := decodeTime(decoder)
	if err != nil {
		return err
	}
	record, err := decoder.DecodeMap()
	if err != nil {
		return err
	}
	if l == 4 {
		option, err := decodeOption(decoder)
		if err != nil {
			return err
		}
		r.Logger.Println("option", option)
	}
	return r.Reader.eventHandler(tag, ts, record)
}

func (r *DefaultMessagesReader) PackedForwardMode(decoder *msgpack.Decoder, tag string, l int) error {
	firstCode, err := decoder.PeekCode()
	if err != nil {
		return err
	}
	var entries []byte
	switch {
	case msgpcode.IsBin(firstCode):
		entries, err = decoder.DecodeBytes()
		if err != nil {
			return err
		}
	case msgpcode.IsString(firstCode):
		return errors.New("PackedForward as string is deprecated")
	}
	var option *Option
	if l == 3 {
		option, err = decodeOption(decoder)
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
		err = r.Ack(option.Chunk)
		if err != nil {
			return err
		}
	}
	for {
		ts, record, err := decodeEntry(_decoder)
		if err != nil {
			if err == io.EOF { // the PackedForward is ended, it's ok.
				return nil
			}
			return err
		}
		err = r.Reader.eventHandler(tag, ts, record)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *DefaultMessagesReader) ForwardMode(decoder *msgpack.Decoder, tag string, l int) error {

	r.Logger.Println("Forward mode")
	size, err := decoder.DecodeArrayLen()
	if err != nil {
		return err
	}
	events := make([]Event, size)
	for i := 0; i < size; i++ {
		ts, record, err := decodeEntry(decoder)
		if err != nil {
			return err
		}
		events[i] = Event{tag, ts, record}
	}

	var option *Option
	if l == 3 { // there is options
		option, err := decodeOption(decoder)
		if err != nil {
			return err
		}
		r.Logger.Println("options", option)
		if option.Chunk != "" {
			r.Logger.Println("ack", option.Chunk)
			err = r.Ack(option.Chunk)
			if err != nil {
				return err
			}
		}
	}

	for _, event := range events {
		err = r.Reader.eventHandler(event.tag, event.ts, event.record)
		if err != nil {
			return err
		}
	}
	//Server SHOULD close the connection silently with no response when the chunk option is not sent.
	if option == nil {
		return io.EOF
	} else {
		if option.Chunk == "" {
			r.Logger.Println("No chunk, so I close the connection.")
			return io.EOF
		}
	}
	return nil
}
