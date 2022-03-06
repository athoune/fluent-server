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

	if msg != "" {
		_list(s.encoder,
			"PONG",
			false, msg,
			s.Hostname,
			hex.EncodeToString(hr.Sum(nil)),
		)
		return fmt.Errorf(msg)
	}
	_list(s.encoder, "PONG",
		true, "",
		s.Hostname,
		hex.EncodeToString(hr.Sum(nil)),
	)
	fmt.Println("< PONG")
	s.step = WaitingForEvents
	return nil
}
