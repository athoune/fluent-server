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
		s = server.NewTLS(m.Handler, cfg)
	} else {
		s = server.New(m.Handler)
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
