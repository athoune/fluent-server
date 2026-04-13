package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.ListenAndServe(l); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	fmt.Printf("\nreceived signal %v, initiating graceful shutdown...\n", sig)

	if err := s.Shutdown(); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("server stopped gracefully")
}
