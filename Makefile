.PHONY: build install

build:
	go build -o ./taragen ./cmd/taragen

install: build
	mv ./taragen /usr/local/bin/taragen

