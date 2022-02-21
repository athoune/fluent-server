package main

import (
	"fmt"
	"os"
	"time"

	"github.com/factorysh/fluent-server/server"
)

func handler(tag string, ts *time.Time, record map[string]interface{}) error {
	fmt.Println(tag, ts, record)
	return nil
}

func main() {
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
		s = server.NewTLS(handler, cfg)
	} else {
		s = server.New(handler)
	}

	l := os.Getenv("LISTEN")
	if l == "" {
		l = "localhost:24224"
	}
	fmt.Println("Listen", l)
	s.ListenAndServe(l)
}
