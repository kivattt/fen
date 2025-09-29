# vim: ts=4 sw=4 ft=make

MAKEFILE_PATH := $(realpath $(lastword $(MAKEFILE_LIST)))
MAKEFILE_DIR  := $(abspath $(dir $(MAKEFILE_PATH)))

# TODO: include internal/benchtest Makefile

# Test options
GO             ?= go
GOBIN           = $(MAKEFILE_DIR)/bin
GO_EXTRA_FLAGS ?=
GO_TEST_FLAGS  ?= -shuffle=on
GO_COVER_MODE  ?= count
ifneq ($(GO_COVER_MODE),)
	GO_COVER_FLAGS = -covermode=$(GO_COVER_MODE)
endif
GO_TEST        ?= $(GO) test $(GO_COVER_FLAGS) $(GO_TEST_FLAGS) $(GO_EXTRA_FLAGS)
GO_GOGC        ?= 800
RICHGO         ?= richgo
RICHGO_VERSION ?= v0.3.12

# Options for linting comments
COMMENTS       ?= 'TODO|WARN|FIXME|CEV'
GREP           ?= \grep
GREP_COLOR     ?= --color=always
GREP_COMMENTS  ?= --line-number --extended-regexp --recursive \
                  --exclude-dir=ucd --exclude-dir=gen --exclude-dir=vendor \
                  --include='*.go' --include=Makefile
xgrep          := $(GREP) $(GREP_COLOR)

# Arguments for `golangci-lint run`
GOLANGCI_VERSION       ?= v1.61.0
GOLANGCI_SORT          ?= --sort-results
GOLANGCI_COLOR         ?= --color=always
GOLANGCI_CONFIG        ?= --config=$(MAKEFILE_DIR)/.golangci.yaml
GOLANGCI_EXTRA_FLAGS   ?=
GOLANGCI_FLAGS         ?= $(GOLANGCI_CONFIG) $(GOLANGCI_SORT) $(GOLANGCI_COLOR) $(GOLANGCI_EXTRA_FLAGS)

# All files related to code generation
GEN_FILES = $(shell find "$(MAKEFILE_DIR)/internal/gen/" \
	-name 'vendor' -prune -o \( -name '*.go' -o -name 'go.*' \) -print)

# Windows exe extension
GEN_TARGET       = $(GOBIN)/gen
GENTABLES_TARGET = $(GOBIN)/gentables
RICHGO_TARGET    = $(GOBIN)/$(RICHGO)
GOLANGCI_TARGET  = $(GOBIN)/golangci-lint
ifeq ($(OS),Windows_NT)
	GEN_TARGET       = $(GEN_TARGET).exe
	GENTABLES_TARGET = $(GENTABLES_TARGET)/gentables
	RICHGO_TARGET    = $(RICHGO_TARGET).exe
	GOLANGCI_TARGET  = $(GOLANGCI_TARGET).exe
endif

# Color support.
red        = $(shell { tput setaf 1 || tput AF 1; } 2>/dev/null)
yellow     = $(shell { tput setaf 3 || tput AF 3; } 2>/dev/null)
cyan       = $(shell { tput setaf 6 || tput AF 6; } 2>/dev/null)
term-reset = $(shell { tput sgr0 || tput me; } 2>/dev/null)

# Install richgo
bin/richgo:
	@if [ ! -x "$(GOBIN)/richgo" ]; then                                                 \
		echo '$(yellow)INFO:$(term-reset) Installing richgo version: $(RICHGO_VERSION)'; \
		mkdir -p $(GOBIN);                                                               \
		GOBIN=$(GOBIN) $(GO) install github.com/kyoh86/richgo@$(RICHGO_VERSION);         \
	fi

# Install golangci-lint
bin/golangci-lint:
	@if [ ! -x "$(GOBIN)/golangci-lint" ]; then                                                    \
		echo '$(yellow)INFO:$(term-reset) Installing golangci-lint version: $(GOLANGCI_VERSION)';  \
		mkdir -p $(GOBIN);                                                                         \
		GOBIN=$(GOBIN) $(GO) install                                                               \
			github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION);               \
	fi

# Build gen
bin/gen: $(MAKEFILE_DIR)/gen.go
	@mkdir -p $(GOBIN)
	@GOBIN=$(GOBIN) $(GO) install -tags=gen gen.go

# Build gentables
bin/gentables: $(GEN_FILES)
	@mkdir -p $(GOBIN)
	@cd $(MAKEFILE_DIR)/internal/gen/gentables && \
		GOBIN=$(GOBIN) $(GO) build -o $(GENTABLES_TARGET)
