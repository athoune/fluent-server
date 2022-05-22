package message

import (
	"crypto/sha512"
	"encoding/hex"

	"github.com/athoune/fluent-server/wire"
)

func (s *FluentSession) doPong(wire *wire.Wire, shared_key_salt, msg string) error {
	err := wire.Encoder.EncodeArrayLen(5)
	if err != nil {
		return err
	}
	err = wire.Encoder.EncodeString("PONG")
	if err != nil {
		return err
	}
	err = wire.Encoder.EncodeBool(msg == "")
	if err != nil {
		return err
	}
	err = wire.Encoder.EncodeString(msg)
	if err != nil {
		return err
	}
	err = wire.Encoder.EncodeString(s.options.Hostname)
	if err != nil {
		return err
	}
	hr := sha512.New()
	hr.Write([]byte(shared_key_salt))
	hr.Write([]byte(s.options.Hostname))
	hr.Write([]byte(s.nonce))
	hr.Write([]byte(s.options.SharedKey))
	err = wire.Encoder.EncodeString(hex.EncodeToString(hr.Sum(nil)))
	if err != nil {
		return err
	}
	err = wire.Flush()
	if err != nil {
		return err
	}
	wire.Debug("< PONG")
	return nil
}
