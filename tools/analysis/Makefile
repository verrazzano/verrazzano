# Copyright (C) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ../../make/quality.mk
include ../../make/generate.mk
include ../../make/retry.mk

MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TOOLS_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/..

NAME:=verrazzano-analysis

LOCAL_VERSION ?= local-$(shell git rev-parse --short HEAD)

OPERATOR_VERSION = ${LOCAL_VERSION}
ANALYSIS_TOOL_VERSION = ${LOCAL_VERSION}
ifdef RELEASE_VERSION
	OPERATOR_VERSION = ${RELEASE_VERSION}
	ANALYSIS_TOOL_VERSION = ${RELEASE_VERSION}
endif
ifndef RELEASE_BRANCH
	RELEASE_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
endif

DIST_DIR:=dist
EXTRA_PARAMS=
INTEG_RUN_ID=
ENV_NAME=verrazzano-analysis
GO = GO111MODULE=on GOPRIVATE=github.com/verrazzano/* go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}

#
#
#
.PHONY: run
run:
	$(GO) run ${GOPATH}/src/github.com/verrazzano/verrazzano/tools/analysis/main.go --zap-log-level=debug

#
# Go build related tasks
#
.PHONY: go-build
go-build:
	GOOS=darwin GOARCH=amd64 $(GO) build \
		-ldflags "${GO_LDFLAGS} -X main.analysisToolVersion=${ANALYSIS_TOOL_VERSION}" \
		-o out/darwin_amd64/verrazzano-analysis \
		${GOPATH}/src/github.com/verrazzano/verrazzano/tools/analysis/main.go
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "${GO_LDFLAGS} -X main.analysisToolVersion=${ANALYSIS_TOOL_VERSION}" \
		-o out/linux_amd64/verrazzano-analysis \
		${GOPATH}/src/github.com/verrazzano/verrazzano/tools/analysis/main.go

.PHONY: go-install
go-install:
	$(GO) install ./...

#
# Test-related tasks
#
.PHONY: unit-test
unit-test: go-install
	$(GO) test -v  ./...
