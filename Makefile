.PHONY: build install-del install web-build web-serve

build:
	go build -o ./taragen ./cmd/taragen

install-del: 
	rm -f $(shell command -v taragen)
install: build
	mv ./taragen /usr/local/bin/taragen


web-build:
	cd web && taragen build

web-serve:
	cd web && taragen serve
