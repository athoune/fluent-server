package message

import (
	"crypto/sha512"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPing(t *testing.T) {
	hashSalt := []byte("salt")

	h := sha512.New()
	h.Write(hashSalt)
	h.Write([]byte("bob"))
	h.Write([]byte("sponge"))

	ping := &Ping{
		client_hostname: "client.local",
		username:        "bob",
		password:        hex.EncodeToString(h.Sum(nil)),
	}

	err := ping.ValidatePassword(hashSalt, func(user string) []byte {
		var passwd []byte
		switch user {
		case "bob":
			passwd = []byte("sponge")
		}
		return passwd
	})
	assert.NoError(t, err)
}
