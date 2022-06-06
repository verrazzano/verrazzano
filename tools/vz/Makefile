# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TOOLS_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/..

NAME:=vz

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
ENV_NAME=vz
GO = GO111MODULE=on GOPRIVATE=github.com/verrazzano/* go
CLI_GO_LDFLAGS ?= -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}

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
		-ldflags "${CLI_GO_LDFLAGS} -X main.cliVersion=${CLI_VERSION}" \
		-o out/darwin_amd64/vz \
		${GOPATH}/src/github.com/verrazzano/verrazzano/tools/vz/main.go
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "${CLI_GO_LDFLAGS} -X main.cliVersion=${CLI_VERSION}" \
		-o out/linux_amd64/vz \
		${GOPATH}/src/github.com/verrazzano/verrazzano/tools/vz/main.go

.PHONY: cli
cli: ## build the CLI
	$(GO) install -ldflags "${CLI_GO_LDFLAGS} -X main.cliVersion=${CLI_VERSION}" ./...

.PHONY: unit-test
unit-test: cli
	$(GO) test -v  ./cmd/...
