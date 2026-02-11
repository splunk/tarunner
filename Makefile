include Makefile.common

CHLOGGEN=chloggen


.PHONY := build
build:
	@mkdir -p bin && cd cmd/tarunner && GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOCMD) build -o ../../bin/tarunner_$(GOOS)_$(GOARCH) .

.PHONY := install-tools
install-tools:
	cd ./internal/tools && $(GOCMD) install go.opentelemetry.io/build-tools/chloggen
	cd ./internal/tools && go install github.com/client9/misspell/cmd/misspell
	cd ./internal/tools && go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint
	cd ./internal/tools && go install github.com/google/addlicense
	cd ./internal/tools && go install golang.org/x/tools/cmd/goimports
	cd ./internal/tools && go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment
	cd ./internal/tools && go install mvdan.cc/gofumpt

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