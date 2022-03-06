package message

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

func (s *FluentSession) handlePing() error {
	fmt.Println("> PING")
	l, err := s.decoder.DecodeArrayLen()
	if err != nil {
		return err
	}
	_type, err := s.decoder.DecodeString()
	if err != nil {
		return err
	}
	if _type != "PING" {
		return fmt.Errorf("wrong type : %s", _type)
	}
	if l != 6 {
		return fmt.Errorf("wrong size for a ping : %v (type='%s')", l, _type)
	}
	ping := make(map[string][]byte)
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
			ping[k] = []byte(v)
		} else {
			if msgpcode.IsBin(code) {
				vv, err := s.decoder.DecodeBytes()
				if err != nil {
					return err
				}
				ping[k] = vv
			} else {
				return fmt.Errorf("unknown type : %v", code)
			}
		}
	}
	s.shared_key_salt = ping["shared_key_salt"]
	for k, v := range ping {
		fmt.Println("ping", k, "=>", string(v))
	}

	// sha512_hex(shared_key_salt + client_hostname + nonce + shared_key)
	shared_key_hexdigest := sha512.New()
	for _, k := range []string{"shared_key_salt", "client_hostname"} {
		shared_key_hexdigest.Write([]byte(ping[k]))
	}
	shared_key_hexdigest.Write([]byte(s.nonce))
	shared_key_hexdigest.Write([]byte(s.SharedKey))
	pingKey := hex.EncodeToString(shared_key_hexdigest.Sum(nil))

	msg := ""
	if string(ping["shared_key_hexdigest"]) != pingKey {
		msg = "shared key mismatch"
	}
	return s.doPong(msg)
}
