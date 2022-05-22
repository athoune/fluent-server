package message

import (
	"crypto/rand"
)

func random(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (s *FluentSession) DoHelo() error {
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
