# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ../make/quality.mk
include ../make/generate.mk

SCRIPT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/../build

NAME:=verrazzano-application-operator
REPO_NAME:=verrazzano-application-operator

CONTROLLER_GEN_VERSION ?= $(shell go list -m -f '{{.Version}}' sigs.k8s.io/controller-tools)
CREATE_LATEST_TAG=0

CRD_OPTIONS ?= "crd:crdVersions=v1"
KUBECONFIG ?= ${HOME}/.kube/config

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

DIST_DIR:=dist
K8S_NAMESPACE:=default
WATCH_NAMESPACE:=
EXTRA_PARAMS=
INTEG_RUN_ID=
ENV_NAME=verrazzano-application-operator
GO ?= CGO_ENABLED=0 GO111MODULE=on GOPRIVATE=github.com/verrazzano go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}

CRD_PATH=../platform-operator/helm_config/charts/verrazzano-application-operator/crds

CLUSTER_DUMP_LOCATION ?= application-operator-integ-cluster-dump

.PHONY: build
build: go-fmt go-vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run:
	# The commands below should be executed to disable the in-cluster verrazzano-application-operator
	# kubectl delete validatingwebhookconfigurations verrazzano-application-ingresstrait-validator
	# kubectl delete mutatingwebhookconfigurations verrazzano-application-istio-defaulter
	# kubectl delete mutatingwebhookconfigurations verrazzano-application-appconfig-defaulter
	# kubectl delete validatingwebhookconfigurations verrazzano-application-verrazzanoproject
	# kubectl scale deployment verrazzano-application-operator --replicas=0 -n verrazzano-system

	# To enable webhooks: $(GO) run main.go --kubeconfig=${KUBECONFIG} --cert-dir=build/webhook-certs
	$(GO) run main.go --kubeconfig=${KUBECONFIG} --enable-webhooks=false --metrics-addr=localhost:0

# Install CRDs into a cluster
.PHONY: install-crds
install-crds: manifests
	kubectl apply -f ${CRD_PATH}

# Uninstall CRDs from a cluster
.PHONY: uninstall-crds
uninstall-crds: manifests
	kubectl delete -f ${CRD_PATH}

# Generate mocks
.PHONY: mock-gen
mock-gen:
	mockgen -destination=mocks/controller_client_mock.go -package=mocks -copyright_file=hack/boilerplate.go.txt sigs.k8s.io/controller-runtime/pkg/client Client,StatusWriter
	mockgen -destination=mocks/controller_manager_mock.go -package=mocks -copyright_file=hack/boilerplate.go.txt sigs.k8s.io/controller-runtime Manager
	mockgen -destination=mocks/logging_scope_handler_mock.go -package=mocks -copyright_file=hack/boilerplate.go.txt github.com/verrazzano/verrazzano/application-operator/controllers/loggingscope Handler
	mockgen -destination=controllers/loggingscope/fluentd_mock.go -package=loggingscope -copyright_file=hack/boilerplate.go.txt github.com/verrazzano/verrazzano/application-operator/controllers/loggingscope FluentdManager

.PHONY: manifests
manifests: application-manifests
#
# Go build related tasks
#
.PHONY: go-build
go-build:
	$(GO) build \
		-ldflags "${GO_LDFLAGS}" \
		-o out/$(shell uname)_$(shell uname -m)/verrazzano-application-operator \
		main.go

.PHONY: go-build-linux
go-build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "${GO_LDFLAGS}" \
		-o out/linux_amd64/verrazzano-application-operator \
		main.go

.PHONY: go-install
go-install:
	$(GO) install

#
# Docker-related tasks
#
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
	$(GO) test -v ./apis/... ./controllers/... ./internal/... ./mcagent/...

#
# Test-related tasks
#
CLUSTER_NAME = apo-integ
VERRAZZANO_NS = verrazzano-system
DEPLOY = deploy
OPERATOR_SETUP = test/operatorsetup
OAM_RUNTIME_IMAGE="ghcr.io/verrazzano/oam-kubernetes-runtime:v0.3.0-20210222205541-9e8d4fb"

# These exports are needed for the install.sh
ifdef JENKINS_KUBECONFIG
export KUBECONFIG=${JENKINS_KUBECONFIG}
export VERRAZZANO_KUBECONFIG=${JENKINS_KUBECONFIG}
else
export KUBECONFIG=${HOME}/.kube/config
export VERRAZZANO_KUBECONFIG=${HOME}/.kube/config
endif

export VERRAZZANO_APP_OP_IMAGE=${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

.PHONY: setup-cluster
setup-cluster: create-cluster
	echo 'Load Docker image for the Verrazzano application operator...'
	time kind load docker-image --name ${CLUSTER_NAME} ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

	echo 'Install OAM runtime and Verrazzano application operator...'
	time installer/install.sh || (echo 'Application operator install failed, capturing kind cluster dump'; ../tools/scripts/k8s-dump-cluster.sh -d ${CLUSTER_DUMP_LOCATION} -r ${CLUSTER_DUMP_LOCATION}/analysis.report; exit 1)

.PHONY: integ-test
integ-test: setup-cluster
	echo 'Run tests...'
	ginkgo -v --keepGoing -cover test/integ/... || (echo 'Application operator tests failed, capturing kind cluster dump'; ../tools/scripts/k8s-dump-cluster.sh -d ${CLUSTER_DUMP_LOCATION} -r ${CLUSTER_DUMP_LOCATION}/analysis.report; exit 1)

.PHONY: cleanup-cluster
cleanup-cluster:
	./build/scripts/cleanup.sh ${CLUSTER_NAME}

.PHONY: create-cluster
create-cluster:
	# We only create a cluster here if this is not on Jenkins. Jenkins will set the cluster
ifndef JENKINS_URL
	kind_clusters=$$(kind get clusters) ; \
    if [ -z "$${kind_clusters}" ] || ! kind get clusters | egrep "^${CLUSTER_NAME}\$$" > /dev/null ; then \
        echo 'Create cluster...' ;\
        echo 'KUBECONFIG=${KUBECONFIG}' ;\
        HTTP_PROXY="" HTTPS_PROXY="" http_proxy="" https_proxy="" time kind create cluster -v 1 \
            --name ${CLUSTER_NAME} \
            --wait 5m \
            --config=test/kind-config.yaml ;\
        kubectl config set-context kind-${CLUSTER_NAME} ;\
    fi
endif

.PHONY: delete-cluster
delete-cluster:
ifdef JENKINS_URL
	echo "Don't bother deleting the cluster on Jenkins, it will not be re-used by anything else"
else
	kind delete cluster --name ${CLUSTER_NAME}
endif

#
# Kubernetes-related tasks
#
.PHONY: push-tag
push-tag:
	PUBLISH_TAG="${DOCKER_IMAGE_TAG}"; \
	echo "Tagging and pushing image ${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	docker pull "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}"; \
	docker tag "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}" "${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	docker push "${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"
