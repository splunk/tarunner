GO?=go
GOOS?=linux
GOARCH?=amd64


.PHONY := test
test:
	@go test -v ./...

build:
	@mkdir -p bin && cd cmd/tarunner && GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o ../../bin/tarunner_$(GOOS)_$(GOARCH) .
