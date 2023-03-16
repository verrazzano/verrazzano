# Copyright (c) 2022, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
VZ_DIR:=github.com$(shell echo ${MAKEFILE_DIR} | sed 's/.*github.com//')
VERSION_DIR:=${VZ_DIR}/cmd/version

NAME:=vz

GIT_COMMIT:=$(shell git rev-parse HEAD)
CLI_VERSION:=$(shell grep verrazzano-development-version ${MAKEFILE_DIR}/../../.verrazzano-development-version | cut -d= -f 2)
BUILD_DATE:=$(shell date +"%Y-%m-%dT%H:%M:%SZ")

ifdef RELEASE_VERSION
	CLI_VERSION=${RELEASE_VERSION}
endif
ifndef RELEASE_BRANCH
	RELEASE_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
endif

DIST_DIR:=dist
ENV_NAME=vz
GO=GO111MODULE=on GOPRIVATE=github.com/verrazzano/* go
CLI_GO_LDFLAGS=-X '${VERSION_DIR}.gitCommit=${GIT_COMMIT}' -X '${VERSION_DIR}.buildDate=${BUILD_DATE}' -X '${VERSION_DIR}.cliVersion=${CLI_VERSION}'

#
# CLI
#

.DEFAULT_GOAL := help
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: run
run:
	$(GO) run ${GOPATH}/src/${VZ_DIR}/main.go
#
# Go build related tasks
#
.PHONY: go-build
go-build: ## build the CLI for multiple architecture and platforms
	GOOS=darwin GOARCH=amd64 $(GO) build \
		-ldflags "${CLI_GO_LDFLAGS}" \
		-o out/darwin_amd64/vz \
		${GOPATH}/src/${VZ_DIR}/main.go
	GOOS=darwin GOARCH=arm64 $(GO) build \
		-ldflags "${CLI_GO_LDFLAGS}" \
		-o out/darwin_arm64/vz \
		${GOPATH}/src/${VZ_DIR}/main.go
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "${CLI_GO_LDFLAGS}" \
		-o out/linux_amd64/vz \
		${GOPATH}/src/${VZ_DIR}/main.go
	GOOS=linux GOARCH=arm64 $(GO) build \
		-ldflags "${CLI_GO_LDFLAGS}" \
		-o out/linux_arm64/vz \
		${GOPATH}/src/${VZ_DIR}/main.go

.PHONY: build-cli
build-cli: ## build the CLI for current system and architecture
	$(GO) build -ldflags "${CLI_GO_LDFLAGS}" -o out/$(shell go env GOOS)-$(shell go env GOARCH)/vz ${GOPATH}/src/${VZ_DIR}/main.go

.PHONY: cli
cli: ## install the CLI
	$(GO) install -ldflags "${CLI_GO_LDFLAGS}" ./...

.PHONY: unit-test
unit-test: cli
	$(GO) test -v  ./...
