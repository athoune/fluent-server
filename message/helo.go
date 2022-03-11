package message

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func random(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (s *FluentSession) doHelo() error {
	var err error
	s.nonce, err = random(16)
	if err != nil {
		return err
	}
	if s.PasswordForKey == nil {
		s.hashSalt = []byte{}
	} else {
		s.hashSalt, err = random(16)
		if err != nil {
			return err
		}
	}
	fmt.Println("< HELO")
	err = s.encoder.EncodeArrayLen(2)
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString("HELO")
	if err != nil {
		return err
	}

	fmt.Printf(`< nonce : %s
  auth: %s
  keepAlive: %v
`, hex.EncodeToString(s.nonce), s.hashSalt, true)

	err = s.encoder.EncodeMapLen(3)
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString("nonce")
	if err != nil {
		return err
	}
	err = s.encoder.EncodeBytes(s.nonce)
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString("auth")
	if err != nil {
		return err
	}
	err = s.encoder.EncodeBytes(s.hashSalt)
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString("keepalive")
	if err != nil {
		return err
	}
	err = s.encoder.EncodeBool(true)
	if err != nil {
		return err
	}
	s.step = WaitingForPing
	return nil
}
