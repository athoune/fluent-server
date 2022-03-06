package message

import (
	"crypto/rand"
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
	err = _map(s.encoder,
		"nonce", s.nonce,
		"auth", s.hashSalt,
		"keepalive", true,
	)
	fmt.Printf(`< nonce : %s
  auth: %s
  keepAlive: %v
`, s.nonce, s.hashSalt, true)
	if err != nil {
		return err
	}
	s.step = WaitingForPing
	return nil
}
