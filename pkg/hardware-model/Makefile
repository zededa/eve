# Copyright (c) 2018 Zededa, Inc.
# SPDX-License-Identifier: Apache-2.0

# Hardware Model Service Makefile
# Extracted from pillar monolith for improved security and maintainability

HOSTARCH      := $(subst aarch64,arm64,$(subst x86_64,amd64,$(shell uname -m)))
ZARCH         ?= $(HOSTARCH)
DISTDIR       := dist/$(ZARCH)
BUILD_VERSION ?= 1.0.0

DOCKER_ARCH_ARGS:=--platform linux/$(ZARCH)
DOCKER_ARGS:=$(DOCKER_ARCH_ARGS) --build-arg ZARCH=$(ZARCH)
DOCKER_TAG:=lfedge/eve-hardware-model:local-$(ZARCH)

SERVICE_NAME := hardware-model

.PHONY: all clean build test build-docker shell

all: build

$(DISTDIR):
	mkdir -p $(DISTDIR)

build: $(DISTDIR)
	@echo "Building $(SERVICE_NAME) for $(ZARCH)"
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ZARCH) go build -ldflags "-X main.version=$(BUILD_VERSION)" -o $(DISTDIR)/$(SERVICE_NAME) ./cmd/$(SERVICE_NAME)

test:
	@echo "Running tests for $(SERVICE_NAME)"
	go test -v ./...

build-docker:
	@echo "Building Docker image for $(SERVICE_NAME)"
	docker build $(DOCKER_ARGS) -t $(DOCKER_TAG) .

clean:
	@echo "Cleaning $(SERVICE_NAME) build artifacts"
	rm -rf $(DISTDIR)
	rm -f $(SERVICE_NAME)

shell:
	docker run --rm -it $(DOCKER_TAG) /bin/sh

run: build
	@echo "Running $(SERVICE_NAME)"
	./$(DISTDIR)/$(SERVICE_NAME) -h

# Development targets
dev-build:
	go build -o $(SERVICE_NAME) ./cmd/$(SERVICE_NAME)

dev-test: dev-build
	./$(SERVICE_NAME) --version
	./$(SERVICE_NAME) -o /tmp/hw-test.txt
	@echo "Hardware model detected:"
	@cat /tmp/hw-test.txt

help:
	@echo "Hardware Model Service Build System"
	@echo ""
	@echo "TARGETS:"
	@echo "  all          Build the service (default)"
	@echo "  build        Build the service binary"
	@echo "  test         Run unit tests"
	@echo "  build-docker Build Docker image"
	@echo "  clean        Clean build artifacts"
	@echo "  shell        Enter Docker container shell"
	@echo "  run          Build and show help"
	@echo "  dev-build    Quick development build"
	@echo "  dev-test     Development test run"
	@echo "  help         Show this help"
	@echo ""
	@echo "OPTIONS:"
	@echo "  ZARCH        Target architecture (amd64, arm64)"
	@echo "  BUILD_VERSION Version to embed in binary"