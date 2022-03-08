package message

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
)

func (s *FluentSession) doPong(msg string) error {
	hr := sha512.New()
	hr.Write([]byte(s.shared_key_salt))
	hr.Write([]byte(s.Hostname))
	hr.Write([]byte(s.nonce))
	hr.Write([]byte(s.SharedKey))

	err := _list(s.encoder,
		"PONG",
		msg == "",
		msg,
		s.Hostname,
		hex.EncodeToString(hr.Sum(nil)),
	)
	fmt.Println("< PONG")

	if err != nil {
		return err
	}
	s.step = WaitingForEvents
	return nil
}
