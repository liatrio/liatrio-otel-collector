# Makefile

CUSTOM_COL_DIR ?= $(CURDIR)/build
OCB_PATH ?= $(CURDIR)/tmp
OCB_VERSION ?= 0.81.0
OCB_URL = https://github.com/open-telemetry/opentelemetry-collector/releases/download/cmd%2Fbuilder%2F
OS := $(shell uname | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m)

# set ARCH var based on output
ifeq ($(ARCH),x86_64)
	ARCH = amd64
endif
ifeq ($(ARCH),aarch64)
	ARCH = arm64
endif

.PHONY: build
build: check_prep
	$(OCB_PATH)/ocb --config testconfig/manifest.yaml

.PHONY: build-debug
build-debug: check_prep
	$(OCB_PATH)/ocb --config testconfig/manifest-debug.yaml

.PHONY: check_prep
check_prep:
	@if [ ! -f $(OCB_PATH)/ocb ]; then make prep; fi

.PHONY: prep
prep:
	@echo "Downloading OpenTelemetry Collector Build"
	@mkdir -p $(OCB_PATH)
	@curl -LO $(OCB_URL)v$(OCB_VERSION)/ocb_$(OCB_VERSION)_$(OS)_$(ARCH)
	@mv ocb_$(OCB_VERSION)_$(OS)_$(ARCH) $(OCB_PATH)/ocb
	@chmod +x $(OCB_PATH)/ocb

.PHONY: run
run: build
	$(CUSTOM_COL_DIR)/otelcol-custom --config testconfig/config.yaml
	
.PHONY: checks
checks:
