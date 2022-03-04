package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/factorysh/fluent-server/mirror"
	"github.com/factorysh/fluent-server/server"
)

func main() {
	m := mirror.New()
	var err error
	var s *server.Server

	caCrt := os.Getenv("CA_CRT")
	if caCrt != "" {
		cfg, err := server.ConfigTLS(caCrt, os.Getenv("SRV_CRT"), os.Getenv("SRV_KEY"))
		if err != nil {
			panic(err)
		}
		fmt.Printf(`
ca.crt: %s
server.crt: %s
server.key: %s
`, caCrt, os.Getenv("SRV_CRT"), os.Getenv("SRV_KEY"))
		s, err = server.NewTLS(m.Handler, cfg)
		if err != nil {
			panic(err)
		}
	} else {
		s, err = server.New(m.Handler)
	}
	if err != nil {
		panic(err)
	}
	sharedKey := os.Getenv("SHARED_KEY")
	if sharedKey != "" {
		s.SharedKey = sharedKey
	}

	ll := os.Getenv("MIRROR_LISTEN")
	if ll == "" {
		ll = "localhost:24280"
	}
	go http.ListenAndServe(ll, m)
	fmt.Println("mirror listen ", ll)

	l := os.Getenv("LISTEN")
	if l == "" {
		l = "localhost:24224"
	}
	fmt.Println("Listen", l)
	s.ListenAndServe(l)
}
