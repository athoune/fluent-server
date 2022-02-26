package message

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
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
}

func New(eventHandler HandlerFunc) *FluentReader {
	return &FluentReader{
		eventHandler: eventHandler,
	}
}

func (f *FluentReader) Listen(ctx context.Context, flux io.ReadWriteCloser) error {
	defer flux.Close()
	decoder := msgpack.NewDecoder(flux)
	encoder := msgpack.NewEncoder(flux)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	var globalError error
	go func() {
		<-ctx.Done()
		flux.Close()
		wg.Done()
	}()

	go func() {
		for {
			err := f.decodeMessage(decoder, encoder)
			if err != nil {
				if err == io.EOF {
					return
				}
				globalError = err
			}
		}
	}()
	wg.Wait()

	return globalError
}

func (f *FluentReader) decodeMessage(decoder *msgpack.Decoder, encoder *msgpack.Encoder) error {
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
		return errors.New("Not an array")
	}
	l, err := decoder.DecodeArrayLen()
	if err != nil {
		return err
	}
	if l == 0 {
		return errors.New("Empty array")
	}
	if l > 10 {
		return errors.New("Flood")
	}
	tag, err := decoder.DecodeString()
	if err != nil {
		return err
	}
	switch tag {
	case "HELO":
		return f.doHelo()
	case "PING":
		return f.doPing()
	case "PONG":
		return f.doPong()
	default:
		return f.decodeEvent(decoder, encoder, tag, l)
	}
}

func decodeEntry(decoder *msgpack.Decoder) (*time.Time, map[string]interface{}, error) {
	c, err := decoder.PeekCode()
	if err != nil {
		return nil, nil, err
	}
	if !msgpcode.IsFixedArray(c) {
		return nil, nil, fmt.Errorf("Not an array : %v", c)
	}
	l, err := decoder.DecodeArrayLen()
	if err != nil {
		return nil, nil, err
	}
	if l != 2 {
		return nil, nil, fmt.Errorf("Bad array length %v", l)
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
			return nil, nil, fmt.Errorf("Unknown ext id %v", id)
		}
		if len != 8 {
			return nil, nil, fmt.Errorf("Unknown ext id size %v", len)
		}
		b := make([]byte, len)
		l, err := decoder.Buffered().Read(b)
		if err != nil {
			return nil, nil, err
		}
		if l != len {
			return nil, nil, fmt.Errorf("Read error, wrong size %v", l)
		}
		// https://pkg.go.dev/mod/github.com/vmihailenco/msgpack/v5@v5.0.0-rc.3#RegisterExt
		sec := binary.BigEndian.Uint32(b)
		usec := binary.BigEndian.Uint32(b[4:])
		ts = time.Unix(int64(sec), int64(usec))

	case msgpcode.IsFixedExt(t):
		fmt.Println("FixedExt")
	default:
		return nil, nil, fmt.Errorf("Unknown type %v", t)
	}
	record, err := decoder.DecodeMap()
	if err != nil {
		return nil, nil, err
	}
	return &ts, record, nil
}

func (f *FluentReader) decodeEvent(decoder *msgpack.Decoder, encoder *msgpack.Encoder, tag string, l int) error {
	if l < 2 {
		return errors.New("Too short")
	}
	firstCode, err := decoder.PeekCode()
	if err != nil {
		return err
	}
	switch {
	case msgpcode.IsFixedArray(firstCode):
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
		var option map[string]interface{}
		var chunk string
		if l == 3 {
			option, err = decoder.DecodeMap()
			if err != nil {
				return err
			}
			fmt.Println("Option", option)
			chunkRaw, ok := option["chunk"]
			if ok {
				chunk, ok = chunkRaw.(string)
				if !ok {
					return fmt.Errorf("Bad chunk type: %v", chunkRaw)
				}
			}
		}
		for _, event := range events {
			err = f.eventHandler(event.tag, event.ts, event.record)
			if err != nil {
				return err
			}
		}
		if chunk != "" {
			err = encoder.Encode(map[string]interface{}{
				"ack": chunk,
			})
			if err != nil {
				return err
			}
		}

	case msgpcode.IsBin(firstCode):
		fmt.Println("Bin")

	case firstCode == msgpcode.Uint32:
		if l > 4 {
			return fmt.Errorf("Message too large: %d", l)
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
		return fmt.Errorf("Bad code %v", firstCode)
	}
	return nil
}

func (f *FluentReader) doHelo() error { return nil }
func (f *FluentReader) doPing() error { return nil }
func (f *FluentReader) doPong() error { return nil }
