# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ../../make/retry.mk

BACKEND_NAME:=psr-backend
MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
PSR_DIR:=${MAKEFILE_DIR}
CLI_DIR:=github.com$(shell echo ${MAKEFILE_DIR} | sed 's/.*github.com//')/psrctl
VERSION_DIR:=$(CLI_DIR)/cmd/version
BACKEND_DIR:=$(PSR_DIR)/backend
DOCKER_REPO:=ghcr.io
DOCKER_REGISTRY=verrazzano

ifndef DOCKER_IMAGE_FULLNAME
DOCKER_IMAGE_NAME ?= ${BACKEND_NAME}
DOCKER_IMAGE_FULLNAME=${DOCKER_IMAGE_NAME}

ifdef DOCKER_REGISTRY
DOCKER_IMAGE_FULLNAME := ${DOCKER_REGISTRY}/${DOCKER_IMAGE_FULLNAME}
endif
ifdef DOCKER_REPO
DOCKER_IMAGE_FULLNAME := ${DOCKER_REPO}/${DOCKER_IMAGE_FULLNAME}
endif
endif
DOCKER_IMAGE_TAG ?= local-$(shell git rev-parse --short HEAD)

GIT_COMMIT:=$(shell git rev-parse HEAD)
BUILD_VERSION:=$(shell grep verrazzano-development-version ${MAKEFILE_DIR}/../../.verrazzano-development-version | cut -d= -f 2)
BUILD_DATE:=$(shell date +"%Y-%m-%dT%H:%M:%SZ")

DIST_DIR:=dist
ENV_NAME=psr
GO=GO111MODULE=on GOPRIVATE=github.com/verrazzano/* CGO_ENABLED=0  go
CLI_GO_LDFLAGS=-X '${VERSION_DIR}.gitCommit=${GIT_COMMIT}' -X '${VERSION_DIR}.buildDate=${BUILD_DATE}' -X '${VERSION_DIR}.cliVersion=${BUILD_VERSION}'

#
# Go build related tasks
#
.PHONY: go-build
go-build:
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags '-extldflags "-static"' \
		-ldflags "-X main.buildVersion=${BUILD_VERSION}" \
		-ldflags "-X main.buildDate=${BUILD_DATE}" \
		-o out/linux_amd64/psr-backend \
		${BACKEND_DIR}/main.go

#
# Go build cli related tasks
#
.PHONY: go-build-cli
go-build-cli:
	GOOS=linux GOARCH=amd64 $(GO) build \
    		-ldflags "${CLI_GO_LDFLAGS}" \
    		-o out/linux_amd64/psrctl \
    		${GOPATH}/src/${CLI_DIR}/main.go

#
# Docker-related tasks
#
.PHONY: docker-clean
docker-clean:
	rm -rf ${DIST_DIR}

.PHONY: docker-build
docker-build: go-build docker-clean
	docker build --pull \
		-t ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG} .

.PHONY: docker-push
docker-push: docker-build
	$(call retry_docker_push,${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG})

.PHONY: kind-load-image
kind-load-image: docker-build
	kind load docker-image --name kind ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}

.PHONY: run-example-oam
run-example-oam: kind-load-image
	helm upgrade --install psr manifests/charts/worker --set imageName=${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}

.PHONY: run-example-k8s
run-example-k8s: kind-load-image
	helm upgrade --install psr manifests/charts/worker --set appType=k8s --set imageName=${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}

#
# run the CLI
#
.PHONY: run
run:
	$(GO) run ${GOPATH}/src/${CLI_DIR}/main.go

#
# build the CLI
#
.PHONY: cli
cli:
	$(GO) install -ldflags "${CLI_GO_LDFLAGS}" ./...

.PHONY: unit-test
unit-test: cli
	$(GO) test -v  ./backend/...
