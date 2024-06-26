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


clean:
	rm -rf venv bin
