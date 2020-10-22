build:
	go build

venv:
	python3 -m venv venv
	./venv/bin/pip install fluent-logger
