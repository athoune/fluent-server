package message

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

type Event struct {
	tag    string
	ts     time.Time
	record map[string]interface{}
}

type HandlerFunc func(tag string, time time.Time, record map[string]interface{}) error

type FluentReader struct {
	eventHandler HandlerFunc
}

func New(eventHandler HandlerFunc) *FluentReader {
	return &FluentReader{
		eventHandler: eventHandler,
	}
}

func (f *FluentReader) Read(reader io.Reader) error {
	decoder := msgpack.NewDecoder(reader)
	for {
		err := f.decodeMessage(decoder)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func (f *FluentReader) decodeMessage(decoder *msgpack.Decoder) error {
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
		return f.decodeEvent(tag, decoder, l)
	}
}

func (f *FluentReader) decodeEvent(tag string, decoder *msgpack.Decoder, l int) error {
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
			c, err := decoder.PeekCode()
			if err != nil {
				return err
			}
			if !msgpcode.IsFixedArray(c) {
				return fmt.Errorf("Elem %d is not an array : %v", i, c)
			}
			l, err := decoder.DecodeArrayLen()
			if err != nil {
				return err
			}
			if l != 2 {
				return fmt.Errorf("Bad array length %v", l)
			}
			t, err := decoder.PeekCode()
			if err != nil {
				return err
			}
			var ts time.Time
			switch {
			case t == msgpcode.Uint32:
				tRaw, err := decoder.DecodeUint32()
				if err != nil {
					return err
				}
				ts = time.Unix(int64(tRaw), 0)
			case msgpcode.IsExt(t):
				fmt.Println("Ext")

			case msgpcode.IsFixedExt(t):
				fmt.Println("FixedExt")
			default:
				return fmt.Errorf("Unknown type %v", t)
			}
			record, err := decoder.DecodeMap()
			if err != nil {
				return err
			}
			events[i] = Event{tag, ts, record}
		}
		var option map[string]interface{}
		if l == 3 {
			option, err = decoder.DecodeMap()
			if err != nil {
				return err
			}
			fmt.Println("Option", option)
		}
		for _, event := range events {
			err = f.eventHandler(event.tag, event.ts, event.record)
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
		return f.eventHandler(tag, time.Unix(int64(ts), 0), record)
	default:
		return fmt.Errorf("Bad code %v", firstCode)
	}
	return nil
}

func (f *FluentReader) doHelo() error { return nil }
func (f *FluentReader) doPing() error { return nil }
func (f *FluentReader) doPong() error { return nil }
