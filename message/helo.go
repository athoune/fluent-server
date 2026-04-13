package message

import (
	"crypto/rand"
	"fmt"
)

func random(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}

func (s *FluentSession) DoHelo() error {
	var err error
	s.nonce, err = random(16)
	if err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}
	if s.PasswordForKey == nil {
		s.hashSalt = []byte{}
	} else {
		s.hashSalt, err = random(16)
		if err != nil {
			return fmt.Errorf("failed to generate hash salt: %w", err)
		}
	}
	s.debug("< HELO")
	err = s.Wire.Encoder.EncodeArrayLen(2)
	if err != nil {
		return err
	}
	err = s.Wire.Encoder.EncodeString("HELO")
	if err != nil {
		return err
	}

	err = s.Wire.Encoder.EncodeMapLen(3)
	if err != nil {
		return err
	}
	err = s.Wire.Encoder.EncodeString("nonce")
	if err != nil {
		return err
	}
	err = s.Wire.Encoder.EncodeBytes(s.nonce)
	if err != nil {
		return err
	}
	err = s.Wire.Encoder.EncodeString("auth")
	if err != nil {
		return err
	}
	err = s.Wire.Encoder.EncodeBytes(s.hashSalt)
	if err != nil {
		return err
	}
	err = s.Wire.Encoder.EncodeString("keepalive")
	if err != nil {
		return err
	}
	err = s.Wire.Encoder.EncodeBool(true)
	if err != nil {
		return err
	}
	err = s.Wire.Flush()
	if err != nil {
		return err
	}
	s.step = WaitingForPing
	return nil
}
