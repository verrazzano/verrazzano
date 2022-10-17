# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

BACKEND_NAME:=psr-backend
MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
PSR_DIR:=github.com$(shell echo ${MAKEFILE_DIR} | sed 's/.*github.com//')
BACKEND_DIR:=$(PSR_DIR)/backend
DOCKER_REPO:=ghcr.io
DOCKER_REGISTRY=verrazzano

ifndef DOCKER_IMAGE_FULLNAME
DOCKER_IMAGE_NAME ?= ${BACKEND_NAME}
DOCKER_IMAGE_FULLNAME=${DOCKER_IMAGE_NAME}
ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),docker-push push-tag))
	ifndef DOCKER_REPO
		$(error DOCKER_REPO must be defined as the name of the Docker repository where image will be pushed)
	endif
	ifndef DOCKER_NAMESPACE
		$(error DOCKER_NAMESPACE must be defined as the name of the Docker namespace where image will be pushed)
	endif
endif
ifdef DOCKER_REGISTRY
DOCKER_IMAGE_FULLNAME := ${DOCKER_REGISTRY}/${DOCKER_IMAGE_FULLNAME}
endif
ifdef DOCKER_REPO
DOCKER_IMAGE_FULLNAME := ${DOCKER_REPO}/${DOCKER_IMAGE_FULLNAME}
endif
endif

DOCKER_IMAGE_TAG ?= local-$(shell git rev-parse --short HEAD)

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

all:
	@echo $(MAKEFILE_DIR)
	@echo $(PSR_DIR)
	@echo $(BACKEND_DIR)
	@echo $(DOCKER_IMAGE_FULLNAME)
	@echo $(DOCKER_IMAGE_TAG)

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

.PHONY: docker-build
docker-build:
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "${CLI_GO_LDFLAGS}" \
		-o out/linux_amd64/psr-backend \
		${GOPATH}/src/${BACKEND_DIR}/main.go

#
# Docker-related tasks
#
.PHONY: docker-clean
docker-clean:
	rm -rf ${DIST_DIR}

.PHONY: docker-build
docker-build: go-build
	# the TPL file needs to be copied into this dir so it is in the docker build context
	cp ../../THIRD_PARTY_LICENSES.txt .
	docker build --pull \
		-t ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG} .

.PHONY: docker-push
docker-push: docker-build
	$(call retry_docker_push,${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG})

.PHONY: load-image
load-image: docker-build
	kind load docker-image --name kind ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}

.PHONY: build-run-example
build-run-example: load-image
	helm upgrade --install psr manifests/helm/k8schart/ --set imageName=${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}

.PHONY: unit-test
unit-test: cli
	$(GO) test -v  ./backend/...
