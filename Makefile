build:
	go build

venv:
	python3 -m venv venv
	./venv/bin/pip install fluent-logger

generate:
	go get github.com/tinylib/msgp
	go generate -v message/message.go
