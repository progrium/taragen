.PHONY: build link web-build web-serve

BIN ?= /usr/local/bin

build:
	go build -o ./local/bin/taragen ./cmd/taragen

link: build
	rm -f $(BIN)/taragen
	ln -s $(PWD)/local/bin/taragen $(BIN)/taragen

web-build:
	cd web && taragen build

web-serve:
	cd web && taragen serve
