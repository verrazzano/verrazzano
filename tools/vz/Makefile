# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

NAME:=vz

GIT_COMMIT = $(shell git rev-parse --short HEAD)
CLI_VERSION = $(shell grep verrazzano-development-version ${MAKEFILE_DIR}/../../.verrazzano-development-version | cut -d= -f 2)
BUILD_DATE = $(shell date '+%Y-%m-%d')

ifdef RELEASE_VERSION
	CLI_VERSION = ${RELEASE_VERSION}
endif
ifndef RELEASE_BRANCH
	RELEASE_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
endif

DIST_DIR:=dist
ENV_NAME=vz
GO = GO111MODULE=on GOPRIVATE=github.com/verrazzano/* go
CLI_GO_LDFLAGS = -X '${MAKEFILE_DIR}/cmd/version.gitCommit=${GIT_COMMIT}' -X '${MAKEFILE_DIR}/cmd/version.buildDate=${BUILD_DATE}' -X '${MAKEFILE_DIR}/cmd/version.cliVersion=${CLI_VERSION}'

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
		-ldflags "${CLI_GO_LDFLAGS}" \
		-o out/darwin_amd64/vz \
		${GOPATH}/src/github.com/verrazzano/verrazzano/tools/vz/main.go
	GOOS=darwin GOARCH=arm64 $(GO) build \
		-ldflags "${CLI_GO_LDFLAGS}" \
		-o out/darwin_arm64/vz \
		${GOPATH}/src/github.com/verrazzano/verrazzano/tools/vz/main.go
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "${CLI_GO_LDFLAGS}" \
		-o out/linux_amd64/vz \
		${GOPATH}/src/github.com/verrazzano/verrazzano/tools/vz/main.go

.PHONY: cli
cli: ## build the CLI
	$(GO) install -ldflags "${CLI_GO_LDFLAGS}" ./...

.PHONY: unit-test
unit-test: cli
	$(GO) test -v  ./cmd/...
