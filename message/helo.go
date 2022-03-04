package message

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func RandomString(size int) (string, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b)[:size], nil
}

func (s *FluentSession) doHelo() error {
	var err error
	s.nonce, err = RandomString(16)
	if err != nil {
		return err
	}
	if s.PasswordForKey == nil {
		s.hashSalt = ""
	} else {
		s.hashSalt, err = RandomString(16)
		if err != nil {
			return err
		}
	}
	fmt.Println("HELO")
	err = s.encoder.EncodeArrayLen(2)
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString("HELO")
	if err != nil {
		return err
	}
	err = _map(s.encoder,
		"nonce", s.nonce,
		"auth", s.hashSalt,
		"keepalive", true,
	)
	if err != nil {
		return err
	}
	s.step = WaitingForPing
	return nil
}
