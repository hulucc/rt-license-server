.PHONY: ca
ca:
	@openssl genrsa -out server.key 2048
	@openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650

.PHONY: build
build:
	@go build -o bin/ github.com/hulucc/rt-license-server/...

.PHONY: run
run: build
	@bin/rt-license-server
