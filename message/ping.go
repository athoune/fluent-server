package message

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/athoune/fluent-server/wire"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

type Ping struct {
	client_hostname      string
	shared_key_salt      []byte
	shared_key_hexdigest string
	username             string
	password             string
}

func decodePing(decoder *msgpack.Decoder) (*Ping, error) {
	p := &Ping{}
	var err error
	p.client_hostname, err = decoder.DecodeString()
	if err != nil {
		return nil, err
	}
	code, err := decoder.PeekCode()
	if err != nil {
		return nil, err
	}
	switch {
	case msgpcode.IsString(code):
		sks, err := decoder.DecodeString()
		if err != nil {
			return nil, err
		}
		p.shared_key_salt = []byte(sks)
	case msgpcode.IsBin(code):
		p.shared_key_salt, err = decoder.DecodeBytes()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("shared_key_salt has an unknown type : %v", code)
	}
	p.shared_key_hexdigest, err = decoder.DecodeString()
	if err != nil {
		return nil, err
	}
	p.username, err = decoder.DecodeString()
	if err != nil {
		return nil, err
	}
	p.password, err = decoder.DecodeString()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Ping) ValidateSharedKeyHexdigest(nonce, sharedKey string) error {
	// sha512_hex(shared_key_salt + client_hostname + nonce + shared_key)
	shared_key_hexdigest := sha512.New()
	shared_key_hexdigest.Write([]byte(p.shared_key_salt))
	shared_key_hexdigest.Write([]byte(p.client_hostname))
	shared_key_hexdigest.Write([]byte(nonce))
	shared_key_hexdigest.Write([]byte(sharedKey))
	if hex.EncodeToString(shared_key_hexdigest.Sum(nil)) == string(p.shared_key_hexdigest) {
		return nil
	}
	return errors.New("shared key mismatch")
}

func (p *Ping) ValidatePassword(hashsalt []byte, user func(string) []byte) error {
	if p.username == "" {
		return errors.New("username is mandatory")
	}
	if p.password == "" {
		return errors.New("password is mandatory")
	}
	pazzword := user(p.username)
	if pazzword == nil || len(p.password) == 0 {
		return fmt.Errorf("unknown user : %s", p.username)
	}
	password := sha512.New()
	password.Write(hashsalt)
	password.Write([]byte(p.username))
	password.Write(pazzword)
	if hex.EncodeToString(password.Sum(nil)) == p.password {
		return nil
	}
	return fmt.Errorf("bad password for user : %s", p.username)
}

func (s *FluentSession) HandlePing(wire *wire.Wire, l int, _type string) error {
	wire.Debug("> PING")
	if _type != "PING" {
		return fmt.Errorf("wrong type : %s", _type)
	}
	if l != 6 {
		return fmt.Errorf("wrong size for a ping : %v (type='%s')", l, _type)
	}

	ping, err := decodePing(wire.Decoder)
	if err != nil {
		return err
	}

	err = ping.ValidateSharedKeyHexdigest(string(s.nonce), s.options.SharedKey)
	msg := ""
	if err != nil {
		s.options.Logger.Printf("Bad shared key digest : %v\n", err)
		msg = err.Error()
	} else if len(s.hashSalt) > 0 {
		err = ping.ValidatePassword(s.hashSalt, s.options.Users)
		if err != nil {
			s.options.Logger.Printf("Bad password : %v\n", err)
		}
		msg = "bad password"
	}

	return s.doPong(wire, string(ping.shared_key_salt), msg)
}
