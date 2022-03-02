package message

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
)

func (s *FluentSession) doPingPong() error {
	fmt.Println("PING")
	l, err := s.decoder.DecodeArrayLen()
	if err != nil {
		return err
	}
	if l != 6 {
		return fmt.Errorf("wrong size : %v", l)
	}
	_type, err := s.decoder.DecodeString()
	if err != nil {
		return err
	}
	if _type != "PING" {
		return fmt.Errorf("wrong type : %s", _type)
	}
	ping := make(map[string]string)
	for _, k := range []string{"client_hostname", "shared_key_salt",
		"shared_key_hexdigest", "username", "password"} {
		v, err := s.decoder.DecodeString()
		if err != nil {
			return err
		}
		ping[k] = v
	}

	h := sha512.New()
	for _, k := range []string{"shared_key_salt", "client_hostname", "nonce"} {
		h.Write([]byte(ping[k]))
	}
	h.Write([]byte(s.SharedKey))
	hr := sha512.New()

	ping["server_hostname"] = "bob.example.com"

	for _, k := range []string{"shared_key_salt", "server_hostname", "nonce"} {
		hr.Write([]byte(ping[k]))
	}
	hr.Write([]byte(s.SharedKey))

	if ping["shared_key_hexdigest"] != hex.EncodeToString(h.Sum(nil)) {
		_list(s.encoder, "PONG",
			false, "shared key mismatch",
			ping["server_hostname"],
			hex.EncodeToString(hr.Sum(nil)),
		)
		return fmt.Errorf("shared key mismatch")
	}
	_list(s.encoder, "PONG",
		true, "",
		ping["server_hostname"],
		hex.EncodeToString(hr.Sum(nil)),
	)

	s.step = WaitingForEvents
	return nil
}
