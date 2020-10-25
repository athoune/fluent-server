package main

import (
	"fmt"

	"github.com/factorysh/fluent-server/server"
)

func main() {

	s := server.New(func(tag string, time uint32, record map[string]interface{}, option map[string]interface{}) error {
		fmt.Println(tag, time, record, option)
		return nil
	})
	s.ListenAndServe("localhost:24224")
}
