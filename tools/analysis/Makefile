# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ../../make/quality.mk
include ../../make/generate.mk

MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

NAME:=verrazzano-analysis
REPO_NAME:=verrazzano-analysis

CREATE_LATEST_TAG=0

ifndef DOCKER_IMAGE_FULLNAME
DOCKER_IMAGE_NAME ?= ${NAME}-dev
DOCKER_IMAGE_FULLNAME=${DOCKER_IMAGE_NAME}
ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),docker-push push-tag))
ifndef DOCKER_REPO
    $(error DOCKER_REPO must be defined as the name of the docker repository where image will be pushed)
endif
ifndef DOCKER_NAMESPACE
    $(error DOCKER_NAMESPACE must be defined as the name of the docker namespace where image will be pushed)
endif
endif
ifdef DOCKER_NAMESPACE
DOCKER_IMAGE_FULLNAME := ${DOCKER_NAMESPACE}/${DOCKER_IMAGE_FULLNAME}
endif
ifdef DOCKER_REPO
DOCKER_IMAGE_FULLNAME := ${DOCKER_REPO}/${DOCKER_IMAGE_FULLNAME}
endif
endif

DOCKER_IMAGE_TAG ?= local-$(shell git rev-parse --short HEAD)

OPERATOR_VERSION = ${DOCKER_IMAGE_TAG}
ifdef RELEASE_VERSION
	OPERATOR_VERSION = ${RELEASE_VERSION}
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
	$(GO) build \
		-ldflags "${GO_LDFLAGS}" \
		-o out/$(shell uname)_$(shell uname -m)/verrazzano-analysis \
		${GOPATH}/src/github.com/verrazzano/verrazzano/tools/analysis/main.go

.PHONY: go-build-linux
go-build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "${GO_LDFLAGS}" \
		-o out/linux_amd64/verrazzano-analysis \
		${GOPATH}/src/github.com/verrazzano/verrazzano/tools/analysis/main.go

.PHONY: go-install
go-install:
	$(GO) install ./...

#
# Docker-related tasks
#
.PHONY: docker-clean
docker-clean:
	rm -rf ${DIST_DIR}

.PHONY: docker-build
docker-build: go-build-linux
	@echo Building verrazzano-analysis image ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
	# the TPL file needs to be copied into this dir so it is in the docker build context
	cp ../../THIRD_PARTY_LICENSES.txt .
	docker build --pull -f Dockerfile \
		-t ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} .

.PHONY: docker-push
docker-push: docker-build
	docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}
	docker push ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}

ifeq ($(CREATE_LATEST_TAG), "1")
	docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:latest;
	docker push ${DOCKER_IMAGE_FULLNAME}:latest;
endif

#
# Test-related tasks
#
.PHONY: unit-test
unit-test: go-install
	$(GO) test -v  ./...

.PHONY: coverage
coverage: unit-test
	./build/scripts/coverage.sh html

.PHONY: push-tag
push-tag:
	PUBLISH_TAG="${DOCKER_IMAGE_TAG}"; \
	echo "Tagging and pushing image ${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	docker pull "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}"; \
	docker tag "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}" "${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	docker push "${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"
