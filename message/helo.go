package message

import (
	"fmt"
)

func (s *FluentSession) doHelo() error {
	fmt.Println("HELO")
	err := s.encoder.EncodeMapLen(2)
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString("type")
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString("HELO")
	if err != nil {
		return err
	}
	err = s.encoder.EncodeString("options")
	if err != nil {
		return err
	}
	return _map(s.encoder,
		"nonce", "",
		"auth", "",
		"keepalive", true,
	)
}
