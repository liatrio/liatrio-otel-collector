include ./Makefile.Common

CUSTOM_COL_DIR ?= $(CURDIR)/build
OCB_PATH ?= $(CURDIR)/tmp
OCB_VERSION ?= 0.91.0
OCB_URL = https://github.com/open-telemetry/opentelemetry-collector/releases/download/cmd%2Fbuilder%2F
OTEL_CONTRIB_REPO = https://github.com/open-telemetry/opentelemetry-collector-contrib.git
OS := $(shell uname | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m)
GORELEASER_VERSION = 1.20.0
GOLANGCI_LINT_VERSION ?= v1.53.2

# Arguments for getting directories & executing commands against them
PKG_DIRS = $(shell find ./* -not -path "./build/*" -not -path "./tmp/*" -type f -name "go.mod" -exec dirname {} \; | sort | grep -E '^./')
CHECKS = generate lint-all test-all tidy-all fmt-all

# set ARCH var based on output
ifeq ($(ARCH),x86_64)
	ARCH = amd64
endif
ifeq ($(ARCH),aarch64)
	ARCH = arm64
endif

.PHONY: build
build: check-prep
	GOOS=$(OS) GOARCH=$(ARCH) $(OCB_PATH)/ocb --config config/manifest.yaml

.PHONY: build-debug
build-debug: check-prep
	sed 's/debug_compilation: false/debug_compilation: true/g' config/manifest.yaml > $(OCB_PATH)/manifest-debug.yaml
	$(OCB_PATH)/ocb --config $(OCB_PATH)/manifest-debug.yaml

.PHONY: release
release: check-prep
	$(OCB_PATH)/ocb --config config/manifest.yaml --skip-compilation
	curl -sfL https://goreleaser.com/static/run | VERSION=v$(GORELEASER_VERSION) DISTRIBUTION=oss bash \
		-s -- --clean --skip-validate --skip-publish --snapshot

.PHONY: check-prep
check-prep:
	if [[ ! -f $(OCB_PATH)/ocb  || ! -d $(OCB_PATH)/opentelemetry-collector-contrib ]]; then make prep; fi

.PHONY: prep
prep:
	echo "Downloading OpenTelemetry Collector Build"
	mkdir -p $(OCB_PATH)
	curl -LO $(OCB_URL)v$(OCB_VERSION)/ocb_$(OCB_VERSION)_$(OS)_$(ARCH)
	mv ocb_$(OCB_VERSION)_$(OS)_$(ARCH) $(OCB_PATH)/ocb
	chmod +x $(OCB_PATH)/ocb
	cd $(OCB_PATH) && git clone --depth 1 $(OTEL_CONTRIB_REPO); \
		cd opentelemetry-collector-contrib && git fetch --depth 1 origin v$(OCB_VERSION) && git checkout FETCH_HEAD;
	cd $(OCB_PATH)/opentelemetry-collector-contrib/cmd/mdatagen && go install .

.PHONY: run
run: build
	$(CUSTOM_COL_DIR)/otelcol-custom --config config/config.yaml

.PHONY: install-tools
install-tools:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install github.com/Khan/genqlient@latest

.PHONY: for-all
for-all:
	@set -e; for dir in $(DIRS); do \
	  (cd "$${dir}" && \
	  	echo "running $${CMD} in $${dir}" && \
	 	$${CMD} ); \
	done
	
.PHONY: lint-all
lint-all:
	$(MAKE) for-all DIRS="$(PKG_DIRS)" CMD="$(MAKE) lint"

.PHONY: generate
generate:
	$(MAKE) for-all DIRS="$(PKG_DIRS)" CMD="$(MAKE) gen"

.PHONY: test-all
test-all:
	$(MAKE) for-all DIRS="$(PKG_DIRS)" CMD="$(MAKE) test"

.PHONY: cibuild
cibuild: check-prep
	$(OCB_PATH)/ocb --config config/manifest.yaml --skip-compilation

.PHONY: dockerbuild
dockerbuild:
	$(MAKE) build OS=linux ARCH=amd64
	docker build . -t liatrio/liatrio-otel-collector:localdev --build-arg BIN_PATH="./build/otelcol-custom"

.PHONY: tidy-all
tidy-all:
	$(MAKE) for-all DIRS="$(PKG_DIRS)" CMD="$(MAKE) tidy"

.PHONY: fmt-all
fmt-all:
	$(MAKE) for-all DIRS="$(PKG_DIRS)" CMD="$(MAKE) fmt"

# Setting the paralellism to 1 to improve output readability. Reevaluate later as needed for performance
.PHONY: checks
checks:
	$(MAKE) -j 1 $(CHECKS)
	@if [ -n "$$(git diff --name-only)" ]; then \
		echo "Some files have changed. Please commit them."; \
		exit 1; \
	else \
		echo "completed successfully."; \
	fi
