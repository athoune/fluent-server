package main

import (
	"fmt"
	"os"
	"time"

	"github.com/factorysh/fluent-server/server"
)

func main() {

	s := server.New(func(tag string, ts *time.Time, record map[string]interface{}) error {
		fmt.Println(tag, ts, record)
		return nil
	})
	l := os.Getenv("LISTEN")
	if l == "" {
		l = "localhost:24224"
	}
	fmt.Println("Listen", l)
	s.ListenAndServe(l)
}
