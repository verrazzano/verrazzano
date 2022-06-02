# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ../../make/quality.mk
include ../../make/generate.mk
include ../../make/retry.mk

MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TOOLS_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/..

NAME:=verrazzano-cli

LOCAL_VERSION ?= local-$(shell git rev-parse --short HEAD)

OPERATOR_VERSION = ${LOCAL_VERSION}
CLI_VERSION = ${LOCAL_VERSION}
ifdef RELEASE_VERSION
	OPERATOR_VERSION = ${RELEASE_VERSION}
	CLI_VERSION = ${RELEASE_VERSION}
endif
ifndef RELEASE_BRANCH
	RELEASE_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
endif

DIST_DIR:=dist
EXTRA_PARAMS=
INTEG_RUN_ID=
ENV_NAME=verrazzano-cli
GO = GO111MODULE=on GOPRIVATE=github.com/verrazzano/* go
GO_LDFLAGS ?= -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}

#
# CLI
#
.PHONY: run
run:
	$(GO) run ${GOPATH}/src/github.com/verrazzano/verrazzano/tools/vz/main.go
#
# Go build related tasks
#
.PHONY: go-build
go-build:
	GOOS=darwin GOARCH=amd64 $(GO) build \
		-ldflags "-X main.cliVersion=${CLI_VERSION}" \
		-o out/darwin_amd64/verrazzano-cli \
		${GOPATH}/src/github.com/verrazzano/verrazzano/tools/vz/main.go
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "-X main.cliVersion=${CLI_VERSION}" \
		-o out/linux_amd64/verrazzano-cli \
		${GOPATH}/src/github.com/verrazzano/verrazzano/tools/vz/main.go

.PHONY: cli
cli: ## build the CLI
	$(GO) install ./...

.PHONY: unit-test
unit-test: cli
	$(GO) test -v  ./cmd/...
