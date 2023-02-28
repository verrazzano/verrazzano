# Copyright (c) 2022, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
VZ_DIR:=github.com$(shell echo ${MAKEFILE_DIR} | sed 's/.*github.com//')
VERSION_DIR:=${VZ_DIR}/cmd/version

NAME:=vz
GO_MIN_VERSION:=1.19.1
RPM_RELEASE_VERSION:=1

ifndef GIT_COMMIT
	GIT_COMMIT=$(shell git rev-parse HEAD)
endif

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
.PHONY: run
run:
	$(GO) run ${GOPATH}/src/${VZ_DIR}/main.go
#
# Go build related tasks
#
.PHONY: go-build
go-build:
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

.PHONY: cli
cli: ## build the CLI
	$(GO) install -ldflags "${CLI_GO_LDFLAGS}" ./...

.PHONY: unit-test
unit-test: cli
	$(GO) test -v  ./...

.PHONY: cli-rpm-source
cli-rpm-source:
	cp ../../go.mod .
	sed -i "s|^module github.com/verrazzano/verrazzano.*|module github.com/verrazzano/verrazzano/tools/vz|" go.mod
	echo "replace github.com/verrazzano/verrazzano => ../../" >> go.mod
	go mod tidy
	go mod vendor
	tar czf /tmp/verrazzano-cli-${CLI_VERSION}.tar.gz -C .. vz
	mkdir -p out
	mv /tmp/verrazzano-cli-${CLI_VERSION}.tar.gz out
	cp vzcli.spec out/verrazzano-cli.spec
	sed -i "s|^%global APP_VERSION.*|%global APP_VERSION ${CLI_VERSION}|" out/verrazzano-cli.spec
	sed -i "s|^%global CLI_COMMIT.*|%global CLI_COMMIT ${GIT_COMMIT}|" out/verrazzano-cli.spec
	sed -i "s|^%global RELEASE_VERSION.*|%global RELEASE_VERSION ${RPM_RELEASE_VERSION}|" out/verrazzano-cli.spec
	sed -i "s|^%global GO_VERSION.*|%global GO_VERSION ${GO_MIN_VERSION}|" out/verrazzano-cli.spec
	rm go.mod
