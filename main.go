package main

import "github.com/factorysh/fluent-server/server"

func main() {

	s := server.New()
	s.ListenAndServe("localhost:24224")
}
