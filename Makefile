include ./Makefile.Common

CUSTOM_COL_DIR ?= $(CURDIR)/build
OCB_PATH ?= $(CURDIR)/tmp
OCB_VERSION ?= 0.81.0
OCB_URL = https://github.com/open-telemetry/opentelemetry-collector/releases/download/cmd%2Fbuilder%2F
OTEL_CONTRIB_REPO = https://github.com/open-telemetry/opentelemetry-collector-contrib.git
OS := $(shell uname | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m)
GORELEASER_VERSION = 1.19.2
GOLANGCI_LINT_VERSION ?= v1.53.2

# Arguments for getting directories & executing commands against them
# PKG_RECEIVER_DIRS = $(shell find ./pkg/receiver/* -type f -name "go.mod" -print -exec dirname {} \; | sort | uniq)
PKG_RECEIVER_DIRS = $(shell find ./pkg/receiver/* -type f -name '*go.mod*' | sed -r 's|/[^/]+$$||' |sort | uniq )

# set ARCH var based on output
ifeq ($(ARCH),x86_64)
	ARCH = amd64
endif
ifeq ($(ARCH),aarch64)
	ARCH = arm64
endif

.PHONY: build
build: check-prep
	$(OCB_PATH)/ocb --config config/manifest.yaml

.PHONY: build-debug
build-debug: check-prep
	$(OCB_PATH)/ocb --config config/manifest-debug.yaml

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

.PHONY: run
run: build
	$(CUSTOM_COL_DIR)/otelcol-custom --config config/config.yaml
	
.PHONY: install-tools
install-tools:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	
.PHONY: lint-all $(PKG_RECEIVER_DIRS)
lint-all: $(PKG_RECEIVER_DIRS)
	
$(PKG_RECEIVER_DIRS):
	$(MAKE) -C $@ lint
	
# Taken from opentelemetry-collector-contrib
.PHONY: for-all
for-all:
	@echo "running $${CMD} in root"
	@$${CMD}
	@set -e; for dir in $(PKG_RECEIVER_DIRS); do \
	@echo "running $${CMD} in $${dir}"
	  (cd "$${dir}" && \
	  	echo "running $${CMD} in $${dir}" && \
	 	$${CMD} ); \
	done

.PHONY: metagen
metagen: check-prep
	@cd tmp/opentelemetry-collector-contrib/cmd/mdatagen && go install .
	@$(MAKE) for-all CMD="go generate ./..."

.PHONY: cibuild
cibuild: check-prep
	$(OCB_PATH)/ocb --config config/manifest.yaml --skip-compilation

.PHONY: dockerbuild
dockerbuild: check-prep
	goreleaser release --snapshot --clean
