build: bin
	go build -o bin/fluent-server

build-linux:
	make build GOOS=linux
	upx bin/fluent-server

bin:
	mkdir -p bin

venv:
	python3 -m venv venv
	./venv/bin/pip install -U pip
	./venv/bin/pip install fluent-logger

test:
	go test -timeout 30s -cover \
		github.com/athoune/fluent-server/message \
		github.com/athoune/fluent-server/server \
		github.com/athoune/fluent-server/wire \
		github.com/athoune/fluent-server/defaultreader

# Functional tests drive the server with a real Fluent Bit container.
# They require a Docker daemon and are skipped without one.
functional-test:
	go test -timeout 10m -v ./functional/...

# Also run the tests that fail on known implementation bugs.
# See the header of functional/forward_test.go.
functional-test-known-bugs:
	FLUENT_TEST_KNOWN_BUGS=1 go test -timeout 10m -v ./functional/...


clean:
	rm -rf venv bin
