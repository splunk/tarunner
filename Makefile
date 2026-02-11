GO?=go
GOOS?=linux
GOARCH?=amd64
CHLOGGEN=chloggen


.PHONY := test
test:
	@go test -v ./...

.PHONY := build
build:
	@mkdir -p bin && cd cmd/tarunner && GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o ../../bin/tarunner_$(GOOS)_$(GOARCH) .

.PHONY := install-tools
install-tools:
	cd ./internal/tools && go install go.opentelemetry.io/build-tools/chloggen

FILENAME?=$(shell git branch --show-current)
.PHONY: chlog-new
chlog-new:
	$(CHLOGGEN) new --filename $(FILENAME)

.PHONY: chlog-validate
chlog-validate:
	$(CHLOGGEN) validate

.PHONY: chlog-preview
chlog-preview:
	$(CHLOGGEN) update --dry

.PHONY: chlog-update
chlog-update:
	$(CHLOGGEN) update -v $(VERSION)