build: bin
	go build -o bin/fluent-server

bin:
	mkdir -p bin

venv:
	python3 -m venv venv
	./venv/bin/pip install fluent-logger

test:
	go test -timeout 30s -cover github.com/factorysh/fluent-server/message
