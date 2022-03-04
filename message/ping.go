package message

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"github.com/vmihailenco/msgpack/v5/msgpcode"
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
		code, err := s.decoder.PeekCode()
		if err != nil {
			return err
		}
		fmt.Println(k, code)
		if msgpcode.IsString(code) {
			v, err := s.decoder.DecodeString()
			if err != nil {
				return err
			}
			ping[k] = v
		} else {
			if msgpcode.IsBin(code) {
				vv, err := s.decoder.DecodeBytes()
				if err != nil {
					return err
				}
				ping[k] = string(vv)
			} else {
				return fmt.Errorf("unknown type : %v", code)
			}
		}
	}
	for k, v := range ping {
		fmt.Println("ping", k, "=>", v)
	}

	// sha512_hex(shared_key_salt + client_hostname + nonce + shared_key)
	shared_key_hexdigest := sha512.New()
	for _, k := range []string{"shared_key_salt", "client_hostname"} {
		shared_key_hexdigest.Write([]byte(ping[k]))
	}
	shared_key_hexdigest.Write([]byte(s.nonce))
	shared_key_hexdigest.Write([]byte(s.SharedKey))
	pingKey := hex.EncodeToString(shared_key_hexdigest.Sum(nil))

	hr := sha512.New()
	ping["server_hostname"] = s.Hostname

	for _, k := range []string{"shared_key_salt", "server_hostname", "nonce"} {
		hr.Write([]byte(ping[k]))
	}
	hr.Write([]byte(s.SharedKey))

	fmt.Println("PONG")
	if ping["shared_key_hexdigest"] != pingKey {
		_list(s.encoder, "PONG",
			false, "shared key mismatch",
			ping["server_hostname"],
			hex.EncodeToString(hr.Sum(nil)),
		)
		return fmt.Errorf("shared key mismatch %v != %v", ping["shared_key_hexdigest"], pingKey)
	}
	_list(s.encoder, "PONG",
		true, "",
		ping["server_hostname"],
		hex.EncodeToString(hr.Sum(nil)),
	)

	s.step = WaitingForEvents
	return nil
}
