.PHONY: build install docs

build:
	go build -o ./taragen ./cmd/taragen

install: build
	mv ./taragen /usr/local/bin/taragen

docs:
	cd docs && taragen serve
