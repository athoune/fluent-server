Fluent Server
=============

[![Tests](https://github.com/athoune/fluent-server/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/athoune/fluent-server/actions/workflows/test.yml)

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
server with the official Fluent clients and assert what the server received
through its HTTP mirror:

    make functional-test

The target passes `-count=1`: those tests are not hermetic — they build
images, start containers and read `functional/testdata` — so Go's test cache
must not replay a previous result. When running them straight from your
editor instead, the harness reads the testdata files so that editing one
invalidates the cache.

Six clients are covered:

 * Fluent Bit, from the `fluent/fluent-bit:5.1.2` image;
 * the official Python logger, `fluent-logger`;
 * the official Node.js logger, `@fluent-org/logger`;
 * the official Ruby logger, `fluent-logger`;
 * the official Java logger, `org.fluentd:fluent-logger`;
 * the official Go logger, `fluent-logger-golang`, used directly as a test
   dependency of this module.

Run a single family with `make functional-test-<family>`, where `<family>` is
one of `fluentbit`, `python`, `javascript`, `ruby`, `java` or `go`:

    make functional-test-python

Apart from the Go client, which speaks plain TCP straight from the test
process, the clients are baked into small images built from
`functional/testdata/`. Docker caches the layers, so only the first run of a
session builds them. Those tests need a Docker daemon; without one they are
skipped.

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
header of `functional/fluentbit_test.go` for the list and the root causes.
