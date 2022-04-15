# Copyright (C) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ../make/quality.mk
include ../make/generate.mk
include ../make/retry.mk

SCRIPT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/../build
TOOLS_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/../tools

NAME:=verrazzano-image-patch-operator
REPO_NAME:=verrazzano-image-patch-operator

CONTROLLER_GEN_VERSION ?= $(shell go list -m -f '{{.Version}}' sigs.k8s.io/controller-tools)
CREATE_LATEST_TAG=0

ifndef DOCKER_IMAGE_FULLNAME
DOCKER_IMAGE_NAME ?= ${NAME}-dev
DOCKER_IMAGE_FULLNAME=${DOCKER_IMAGE_NAME}
ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),docker-push push-tag))
	ifndef DOCKER_REPO
		$(error DOCKER_REPO must be defined as the name of the Docker repository where image will be pushed)
	endif
	ifndef DOCKER_NAMESPACE
		$(error DOCKER_NAMESPACE must be defined as the name of the Docker namespace where image will be pushed)
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

CRD_OPTIONS ?= "crd:crdVersions=v1"

DIST_DIR:=dist
K8S_NAMESPACE:=default
WATCH_NAMESPACE:=
EXTRA_PARAMS=
INTEG_RUN_ID=
ENV_NAME=verrazzano-image-patch-operator
GO ?= CGO_ENABLED=0 GO111MODULE=on GOPRIVATE=github.com/verrazzano go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}
KIND_CONFIG ?= kind-config.yaml
CRD_PATH=../image-patch-operator/helm_config/charts/image-patch-operator/crds

CLUSTER_DUMP_LOCATION ?= image-patch-operator-integ-cluster-dump

.PHONY: go-build
go-build:
	$(GO) build \
		-ldflags "${GO_LDFLAGS}" \
		-o out/$(shell uname)_$(shell uname -m)/verrazzano-image-patch-operator \
		main.go

.PHONY: go-build-linux
go-build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "${GO_LDFLAGS}" \
		-o out/linux_amd64/verrazzano-image-patch-operator \
		main.go

.PHONY: go-install
go-install:
	$(GO) install

.PHONY: docker-clean
docker-clean:
	rm -rf ${DIST_DIR}

.PHONY: docker-build
docker-build: go-build-linux
	# the TPL file needs to be copied into this dir so it is in the docker build context
	cp ../THIRD_PARTY_LICENSES.txt .
	docker build --pull \
		-t ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} .

.PHONY: docker-push
docker-push: docker-build
	docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}
	$(call retry_docker_push,${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG})

ifeq ($(CREATE_LATEST_TAG), "1")
	docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:latest;
	$(call retry_docker_push,${DOCKER_IMAGE_FULLNAME}:latest);
endif

#
# Test-related tasks
#
.PHONY: unit-test
unit-test: go-install
	$(GO) test -v ./api/... ./controllers/...

# Generate mocks
.PHONY: mock-gen
mock-gen:
	mockgen -destination=mocks/controller_client_mock.go -package=mocks -copyright_file=hack/boilerplate.go.txt sigs.k8s.io/controller-runtime/pkg/client Client,StatusWriter
	mockgen -destination=mocks/controller_manager_mock.go -package=mocks -copyright_file=hack/boilerplate.go.txt sigs.k8s.io/controller-runtime Manager

.PHONY: manifests
manifests: image-patch-manifests

#
# Kubernetes-related tasks
#
.PHONY: push-tag
push-tag:
	PUBLISH_TAG="${DOCKER_IMAGE_TAG}"; \
	echo "Tagging and pushing image ${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	docker pull "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}"; \
	docker tag "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}" "${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	$(call retry_docker_push,"${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG")
