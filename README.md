Fluent Server
=============

[![Build Status](https://drone.garambrogne.net/api/badges/athoune/fluent-server/status.svg)](https://drone.garambrogne.net/athoune/fluent-server)

Listen events coming from fluent-bit or any fluentd hoses.

Features
--------

See https://github.com/fluent/fluentd/wiki/Forward-Protocol-Specification-v1

 * [x] Hearthbeat
 * [x] ack
 * [x] mTls authentication
 * [x] shared key
 * [x] message mode
 * [x] forward mode
 * [x] packed forward
 * [x] compressed packed forward
 * [ ] UDP subprotocol
 * [ ] TlS
 * [ ] username/password authentication

Functional tests
----------------

`make test` runs the unit tests only. The tests in `functional/` drive the
server with a real Fluent Bit client running in a container, and assert what
the server received through its HTTP mirror:

    make functional-test

They need a Docker daemon and pull `fluent/fluent-bit:5.1.2`; without Docker
they are skipped.

By default the harness starts the server in-process, on ephemeral ports: a
single `go test` is enough, and breakpoints in `server/`, `message/` and
`wire/` are hit while debugging the test in VSCode.

To debug the server itself, launch it in the debugger (`Run and Debug` →
`Go: Launch Package` on `main.go`, or `go run .`) with

    LISTEN=0.0.0.0:24224 MIRROR_LISTEN=127.0.0.1:24280 ./bin/fluent-server

then run the functional tests against it:

    FLUENT_TEST_MIRROR_URL=http://127.0.0.1:24280 go test ./functional/...

`LISTEN` must be reachable from the container, hence `0.0.0.0`. In this mode,
tests needing control over the server configuration (shared key, TLS) are
skipped.

Some functional tests fail on known implementation bugs and are skipped by
default; `make functional-test-known-bugs` runs them (they will fail). See the
header of `functional/forward_test.go` for the list and the root causes.
