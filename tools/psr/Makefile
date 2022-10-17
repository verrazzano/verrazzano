# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
PSR_DIR:=github.com$(shell echo ${MAKEFILE_DIR} | sed 's/.*github.com//')
BACKEND_DIR:=$(PSR_DIR)/backend

all:
	@echo $(MAKEFILE_DIR)
	@echo $(PSR_DIR)
	@echo $(BACKEND_DIR)

NAME:=psr

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
ENV_NAME=psr
GO=GO111MODULE=on GOPRIVATE=github.com/verrazzano/* go

#
# PSR Backend
#
.PHONY: run-backend
run-backend:
	$(GO) run ${GOPATH}/src/${BACKEND_DIR}/main.go
#
# Go build related tasks
#
.PHONY: go-build
go-build:
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "${CLI_GO_LDFLAGS}" \
		-o out/linux_amd64/psr-backend \
		${GOPATH}/src/${BACKEND_DIR}/main.go

.PHONY: unit-test
unit-test: cli
	$(GO) test -v  ./backend/...
