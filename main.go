package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/athoune/fluent-server/defaultreader"
	"github.com/athoune/fluent-server/mirror"
	"github.com/athoune/fluent-server/options"
	"github.com/athoune/fluent-server/server"
)

func main() {
	m := mirror.New()
	var err error
	var s *server.Server
	config := &options.FluentOptions{
		MessagesReaderFactory: defaultreader.DefaultMessagesReaderFactory(m.Handler),
	}
	sharedKey := os.Getenv("SHARED_KEY")
	if sharedKey != "" {
		config.SharedKey = sharedKey
	}

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
		s, err = server.NewTLS(config, cfg)
		if err != nil {
			panic(err)
		}
	} else {
		s, err = server.New(config)
	}
	if err != nil {
		panic(err)
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
	err = s.ListenAndServe(l)
	if err != nil {
		panic(err)
	}
}
