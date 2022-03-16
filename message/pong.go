package message

import (
	"crypto/sha512"
	"encoding/hex"
)

func (s *FluentSession) doPong(shared_key_salt, msg string) error {
	err := s.encoder.EncodeArrayLen(5)
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString("PONG")
	if err != nil {
		return err
	}
	err = s.encoder.EncodeBool(msg == "")
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString(msg)
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString(s.Hostname)
	if err != nil {
		return err
	}
	hr := sha512.New()
	hr.Write([]byte(shared_key_salt))
	hr.Write([]byte(s.Hostname))
	hr.Write([]byte(s.nonce))
	hr.Write([]byte(s.SharedKey))
	err = s.encoder.EncodeString(hex.EncodeToString(hr.Sum(nil)))
	if err != nil {
		return err
	}
	err = s.Flush()
	if err != nil {
		return err
	}
	s.debug("< PONG")

	s.step = WaitingForEvents
	return nil
}
