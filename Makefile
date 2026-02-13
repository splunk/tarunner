include Makefile.common

CHLOGGEN=chloggen

.PHONY := build
build:
	mkdir -p bin && cd cmd/tarunner && GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOCMD) build -o ../../bin/tarunner_$(GOOS)_$(GOARCH) .

.PHONY := install-tools
install-tools:
	cd ./internal/tools && $(GOCMD) install go.opentelemetry.io/build-tools/chloggen
	cd ./internal/tools && $(GOCMD) install github.com/client9/misspell/cmd/misspell
	cd ./internal/tools && $(GOCMD) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint
	cd ./internal/tools && $(GOCMD) install github.com/google/addlicense
	cd ./internal/tools && $(GOCMD) install golang.org/x/tools/cmd/goimports
	cd ./internal/tools && $(GOCMD) install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment
	cd ./internal/tools && $(GOCMD) install mvdan.cc/gofumpt

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

.PHONY: windows_amd64-windows_arm64-linux_amd64-linux_arm64-darwin_amd64-darwin_arm64-linux_ppc64le-aix_ppc64_build
%_build:
	$(eval OS:=$(word 1,$(subst _, ,$@)))
	$(eval ARCH:=$(word 2,$(subst _, ,$@)))
	mkdir -p bin && cd cmd/tarunner && GOOS=$(OS) GOARCH=$(ARCH) $(GOCMD) build -o ../../bin/tarunner_$(OS)_$(ARCH) .

package: windows_amd64_build windows_arm64_build linux_amd64_build linux_arm64_build darwin_amd64_build darwin_arm64_build linux_ppc64le_build aix_ppc64_build