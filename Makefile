build: bin
	go build -o bin/fluent-server

build-linux:
	make build GOOS=linux
	upx bin/fluent-server

bin:
	mkdir -p bin

test:
	go test -timeout 30s -cover \
		github.com/athoune/fluent-server/message \
		github.com/athoune/fluent-server/server \
		github.com/athoune/fluent-server/wire \
		github.com/athoune/fluent-server/defaultreader

# Functional tests drive the server with real client containers. They are not
# hermetic and depend on files under functional/testdata, so -count=1 keeps
# them out of Go's test cache, which would otherwise replay a stale result.
# They require a Docker daemon and are skipped without one.
functional-test:
	go test -count=1 -timeout 10m -v ./functional/...

# Also run the tests that fail on known implementation bugs.
# See the header of functional/fluentbit_test.go.
functional-test-known-bugs:
	FLUENT_TEST_KNOWN_BUGS=1 go test -count=1 -timeout 10m -v ./functional/...

clean:
	rm -rf bin
