package message

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"sync"
	"testing"

	"github.com/athoune/fluent-server/msg"
	"github.com/athoune/fluent-server/options"
	"github.com/athoune/fluent-server/wire"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

type Client struct {
	Encoder *msgpack.Encoder
	Decoder *msgpack.Decoder
}

func NewClient(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Client{
		Encoder: msgpack.NewEncoder(conn),
		Decoder: msgpack.NewDecoder(conn),
	}, nil
}

func mockupServer(ctx context.Context, opts *options.FluentOptions) (*Client, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	client, err := NewClient(listener.Addr().String())
	if err != nil {
		return nil, err
	}
	go func() {
		<-ctx.Done()
		listener.Close()
	}()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				panic(err)
			}
			session := NewSession(opts, conn)
			go session.Loop()
		}
	}()
	return client, nil
}

type DummyMessagesReader struct {
	wg     *sync.WaitGroup
	record map[string]interface{}
}

func drain(wire *wire.Wire, value interface{}) error {
	err := wire.Decoder.Decode(&value)
	if err != nil {
		return err
	}
	spew.Dump("Draining :", value)

	return nil
}

func (d *DummyMessagesReader) MessageMode(wire *wire.Wire, tag string) error {
	d.wg.Done()
	fmt.Println("forward mode", tag)
	var v interface{}
	return drain(wire, &v)
}

func (d *DummyMessagesReader) PackedForwardMode(tag string, blob []byte, opt *msg.Option) error {
	return nil
}

func (d *DummyMessagesReader) ForwardMode(wire *wire.Wire, tag string) error {
	var v [][]interface{}
	err := drain(wire, &v)
	if err != nil {
		return err
	}
	for i := 0; i < len(v); i++ {
		// v[i][0] is a timestamp

		for key, value := range v[i][1].(map[string]interface{}) {
			d.record[key] = value
		}
		d.wg.Done()
	}
	return nil
}

func DummyMessagesReaderFactory(wg *sync.WaitGroup, record map[string]interface{}) options.MessagesReaderFactory {
	return func(log *log.Logger, cfg map[string]interface{}) options.MessagesReader {
		return &DummyMessagesReader{
			wg:     wg,
			record: record,
		}
	}
}

func TestSession(t *testing.T) {
	wg := &sync.WaitGroup{}
	myRecord := make(map[string]interface{})
	opt := &options.FluentOptions{
		Logger:                log.Default(),
		MessagesReaderFactory: DummyMessagesReaderFactory(wg, myRecord),
	}
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	client, err := mockupServer(ctx, opt)
	assert.NoError(t, err)

	wg.Add(1)

	err = client.Encoder.Encode([]interface{}{
		"tag.name",
		[]interface{}{
			[]interface{}{1441588984, map[string]interface{}{
				"message": "foo",
			}},
		},
		map[string]interface{}{
			"chunk": "p8n9gmxTQVC8/nh2wlKKeQ==",
			"size":  1,
		},
	})
	assert.NoError(t, err)

	wg.Wait()
	ack, err := client.Decoder.DecodeMap()
	assert.NoError(t, err)
	assert.Equal(t, "p8n9gmxTQVC8/nh2wlKKeQ==", ack["ack"])
	assert.Equal(t, "foo", myRecord["message"])

}

func TestSessionSharedKey(t *testing.T) {
	wg := &sync.WaitGroup{}
	myRecord := make(map[string]interface{})
	const shared_key = "beuha"
	opt := &options.FluentOptions{
		Logger:                log.Default(),
		SharedKey:             shared_key,
		Hostname:              "server.example.com",
		MessagesReaderFactory: DummyMessagesReaderFactory(wg, myRecord),
	}
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	client, err := mockupServer(ctx, opt)
	assert.NoError(t, err)
	assert.NoError(t, err)

	l, err := client.Decoder.DecodeArrayLen()
	assert.NoError(t, err)
	assert.Equal(t, 2, l)
	_type, err := client.Decoder.DecodeString()
	assert.NoError(t, err)
	assert.Equal(t, "HELO", _type)
	options, err := client.Decoder.DecodeMap()
	assert.NoError(t, err)
	nonce := options["nonce"].([]byte)
	auth := options["auth"].([]byte)
	assert.Equal(t, []byte{}, auth, "No auth")
	// sha512_hex(shared_key_salt + client_hostname + nonce + shared_key)
	h := sha512.New()
	h.Write([]byte("my_salt"))
	h.Write([]byte("client.example.com"))
	h.Write([]byte(nonce))
	h.Write([]byte(shared_key))
	err = client.Encoder.Encode([]string{
		"PING",
		"client.example.com",
		"my_salt",
		hex.EncodeToString(h.Sum(nil)),
		"",
		"",
	})
	assert.NoError(t, err)

	l, err = client.Decoder.DecodeArrayLen()
	assert.NoError(t, err)
	assert.Equal(t, 5, l)
	_type, err = client.Decoder.DecodeString()
	assert.NoError(t, err)
	assert.Equal(t, "PONG", _type)
	auth_result, err := client.Decoder.DecodeBool()
	assert.NoError(t, err)
	assert.True(t, auth_result)
	reason, err := client.Decoder.DecodeString()
	assert.NoError(t, err)
	assert.Equal(t, "", reason)
	server_hostname, err := client.Decoder.DecodeString()
	assert.NoError(t, err)
	assert.Equal(t, "server.example.com", server_hostname)

	wg.Add(1)

	err = client.Encoder.Encode([]interface{}{
		"tag.name",
		[]interface{}{
			[]interface{}{1441588984, map[string]interface{}{
				"message": "foo",
			}},
		},
		map[string]interface{}{
			"chunk": "p8n9gmxTQVC8/nh2wlKKeQ==",
			"size":  1,
		},
	})
	assert.NoError(t, err)

	wg.Wait()
	assert.Equal(t, "foo", myRecord["message"])
}
