include ./Makefile.Common

CUSTOM_COL_DIR ?= $(SRC_ROOT)/build
TMP_DIR ?= $(SRC_ROOT)/tmp
OS := $(shell uname | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m)

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
build: install-tools
	GOOS=$(OS) GOARCH=$(ARCH) $(OCB) --config config/manifest.yaml

.PHONY: build-debug
build-debug: $(TMP_DIR) install-tools
	sed 's/debug_compilation: false/debug_compilation: true/g' config/manifest.yaml > $(TMP_DIR)/manifest-debug.yaml
	$(OCB) --config $(TMP_DIR)/manifest-debug.yaml

.PHONY: release
release:
	$(OCB) --config config/manifest.yaml --skip-compilation
	$(GORELEASER) --clean --skip-validate --skip-publish --snapshot

$(TMP_DIR):
	mkdir -p $@

.PHONY: run
run: build
	$(CUSTOM_COL_DIR)/otelcol-custom --config config/config.yaml

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
cibuild: install-tools
	$(OCB) --config config/manifest.yaml --skip-compilation

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
