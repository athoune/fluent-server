package message

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

type Event struct {
	tag    string
	ts     *time.Time
	record map[string]interface{}
}

type HandlerFunc func(tag string, time *time.Time, record map[string]interface{}) error

type FluentReader struct {
	eventHandler HandlerFunc
	auth         bool
}

func New(eventHandler HandlerFunc) *FluentReader {
	return &FluentReader{
		eventHandler: eventHandler,
	}
}

func (f *FluentReader) helo(encoder msgpack.Encoder, decoder msgpack.Decoder) error {

	err := encoder.EncodeMapLen(2)
	if err != nil {
		return err
	}
	return nil
}

func (f *FluentReader) Listen(ctx context.Context, flux io.ReadWriteCloser) error {
	defer flux.Close()
	decoder := msgpack.NewDecoder(flux)
	encoder := msgpack.NewEncoder(flux)
	encoder.UseCompactInts(true)
	encoder.UseCompactFloats(true)
	go func() {
		<-ctx.Done()
		flux.Close()
	}()

	for {
		err := f.handleMessage(decoder, encoder)
		if err != nil {
			return err
		}
	}
}

func (f *FluentReader) handleMessage(decoder *msgpack.Decoder, encoder *msgpack.Encoder) error {
	code, err := decoder.PeekCode()
	if err != nil {
		return err
	}
	if code == msgpcode.Nil {
		err = decoder.DecodeNil()
		if err != nil {
			return err
		}
		fmt.Println("Hearthbeat")
		return nil
	}
	if !msgpcode.IsFixedArray(code) {
		return errors.New("not an array")
	}
	l, err := decoder.DecodeArrayLen()
	if err != nil {
		return err
	}
	if l == 0 {
		return errors.New("empty array")
	}
	if l > 5 {
		return errors.New("flood")
	}
	_type, err := decoder.DecodeString()
	if err != nil {
		return err
	}
	fmt.Println("message type", _type)
	switch _type {
	case "PING":
		return f.doPing()
	default: // It's a tag
		return f.decodeMessages(decoder, encoder, _type, l)
	}
}

func (f *FluentReader) decodeMessages(decoder *msgpack.Decoder, encoder *msgpack.Encoder, tag string, l int) error {
	if l < 2 {
		return errors.New("too short")
	}
	firstCode, err := decoder.PeekCode()
	if err != nil {
		return err
	}
	switch {
	case msgpcode.IsFixedArray(firstCode): // Forward mode
		fmt.Println("Forward mode")
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
		if l == 3 {
			var chunk string
			var key string
			option_l, err := decoder.DecodeMapLen()
			if err != nil {
				return err
			}
			for i := 0; i < option_l; i++ {
				key, err = decoder.DecodeString()
				if err != nil {
					return err
				}
				fmt.Println("option key :", key)
				switch key {
				case "chunk":
					chunk, err = decoder.DecodeString()
				default:
					_, err = decoder.DecodeInterface()
				}
				if err != nil {
					return err
				}
			}
			if chunk != "" {
				fmt.Println("ack", chunk)
				err = encoder.EncodeMapLen(1)
				if err != nil {
					return err
				}
				err = encoder.EncodeString("ack")
				if err != nil {
					return err
				}
				err = encoder.EncodeString(chunk)
				if err != nil {
					return err
				}
			}
		}
		for _, event := range events {
			err = f.eventHandler(event.tag, event.ts, event.record)
			if err != nil {
				return err
			}
		}

	case msgpcode.IsBin(firstCode) || msgpcode.IsString(firstCode): // PackedForward Mode
		fmt.Println("PackedForward Mode")
		fmt.Println("Bin")

	case firstCode == msgpcode.Uint32: // Message Mode
		fmt.Println("Message Mode")
		if l > 4 {
			return fmt.Errorf("message too large: %d", l)
		}
		ts, err := decoder.DecodeUint32()
		if err != nil {
			return err
		}
		record, err := decoder.DecodeMap()
		if err != nil {
			return err
		}
		if l == 4 {
			option, err := decoder.DecodeMap()
			if err != nil {
				return err
			}
			fmt.Println("option", option)
		}
		tz := time.Unix(int64(ts), 0)
		return f.eventHandler(tag, &tz, record)
	default:
		return fmt.Errorf("bad code %v", firstCode)
	}
	return nil
}
func decodeEntry(decoder *msgpack.Decoder) (*time.Time, map[string]interface{}, error) {
	c, err := decoder.PeekCode()
	if err != nil {
		return nil, nil, err
	}
	if !msgpcode.IsFixedArray(c) {
		return nil, nil, fmt.Errorf("not an array : %v", c)
	}
	l, err := decoder.DecodeArrayLen()
	if err != nil {
		return nil, nil, err
	}
	if l != 2 {
		return nil, nil, fmt.Errorf("bad array length %v", l)
	}
	t, err := decoder.PeekCode()
	if err != nil {
		return nil, nil, err
	}
	var ts time.Time
	switch {
	case t == msgpcode.Uint32:
		tRaw, err := decoder.DecodeUint32()
		if err != nil {
			return nil, nil, err
		}
		ts = time.Unix(int64(tRaw), 0)
	case msgpcode.IsExt(t):
		id, len, err := decoder.DecodeExtHeader()
		if err != nil {
			return nil, nil, err
		}
		if id != 0 {
			return nil, nil, fmt.Errorf("unknown ext id %v", id)
		}
		if len != 8 {
			return nil, nil, fmt.Errorf("unknown ext id size %v", len)
		}
		b := make([]byte, len)
		l, err := decoder.Buffered().Read(b)
		if err != nil {
			return nil, nil, err
		}
		if l != len {
			return nil, nil, fmt.Errorf("read error, wrong size %v", l)
		}
		// https://pkg.go.dev/mod/github.com/vmihailenco/msgpack/v5@v5.0.0-rc.3#RegisterExt
		sec := binary.BigEndian.Uint32(b)
		usec := binary.BigEndian.Uint32(b[4:])
		ts = time.Unix(int64(sec), int64(usec))

	case msgpcode.IsFixedExt(t):
		fmt.Println("FixedExt")
	default:
		return nil, nil, fmt.Errorf("unknown type %v", t)
	}
	record, err := decoder.DecodeMap()
	if err != nil {
		return nil, nil, err
	}
	return &ts, record, nil
}

func (f *FluentReader) doHelo() error { return nil }
func (f *FluentReader) doPing() error { return nil }
func (f *FluentReader) doPong() error { return nil }
