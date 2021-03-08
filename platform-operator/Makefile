# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ../make/quality.mk
include ../make/generate.mk

NAME:=verrazzano-platform-operator
REPO_NAME:=verrazzano-platform-operator

TIMESTAMP := $(shell date -u +%Y%m%d%H%M%S)
DOCKER_IMAGE_NAME ?= ${NAME}-dev
DOCKER_IMAGE_TAG ?= local-${TIMESTAMP}-$(shell git rev-parse --short HEAD)

CONTROLLER_GEN_VERSION ?= $(shell go list -m -f '{{.Version}}' sigs.k8s.io/controller-tools)

CREATE_LATEST_TAG=0

CRD_OPTIONS ?= "crd:crdVersions=v1"

ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),docker-push push-tag))
ifndef DOCKER_REPO
    $(error DOCKER_REPO must be defined as the name of the docker repository where image will be pushed)
endif
ifndef DOCKER_NAMESPACE
    $(error DOCKER_NAMESPACE must be defined as the name of the docker namespace where image will be pushed)
endif
DOCKER_IMAGE_FULLNAME = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${DOCKER_IMAGE_NAME}
endif

ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),docker-build docker-push))
ifndef VERRAZZANO_APPLICATION_OPERATOR_IMAGE
    $(error VERRAZZANO_APPLICATION_OPERATOR_IMAGE must be defined as the application operator image that will be installed by the platform operator)
endif
endif

OPERATOR_VERSION = ${DOCKER_IMAGE_TAG}
ifdef RELEASE_VERSION
	OPERATOR_VERSION = ${RELEASE_VERSION}
endif
ifndef RELEASE_BRANCH
	RELEASE_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
endif

export KUBECONFIG=${HOME}/.kube/vpo-integ-config
export VERRAZZANO_KUBECONFIG=${HOME}/.kube/vpo-integ-config

DIST_DIR:=dist
K8S_NAMESPACE:=default
WATCH_NAMESPACE:=
EXTRA_PARAMS=
INTEG_RUN_ID=
ENV_NAME=verrazzano-platform-operator
GO ?= GO111MODULE=on GOPRIVATE=github.com/verrazzano go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}

CRD_PATH=config/crd/bases
CODEGEN_PATH = k8s.io/code-generator

# Run against the configured Kubernetes cluster specified by KUBECONFIG
.PHONY: run
run:
	$(GO) run main.go --kubeconfig=${KUBECONFIG} --zap-log-level=debug

# Install CRDs into a cluster
.PHONY: install-crds
install-crds:
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
.PHONY: uninstall-crds
uninstall-crds:
	kustomize build config/crd | kubectl delete -f -


#
# Go build related tasks
#
.PHONY: go-build
go-build:
	$(GO) build \
		-ldflags "${GO_LDFLAGS}" \
		-o out/$(shell uname)_$(shell uname -m)/verrazzano-platform-operator \
		main.go

.PHONY: go-build-linux
go-build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "${GO_LDFLAGS}" \
		-o out/linux_amd64/verrazzano-platform-operator \
		main.go

.PHONY: go-install
go-install:
	$(GO) install ./...

.PHONY: manifests
manifests: platform-manifests
#
# Docker-related tasks
#
.PHONY: docker-clean
docker-clean:
	rm -rf ${DIST_DIR}

.PHONY: docker-build
docker-build: go-build-linux
	@echo Building verrazzano-platform-operator image ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
	@echo using verrazzano-application-operator image ${VERRAZZANO_APPLICATION_OPERATOR_IMAGE}
	# the TPL file needs to be copied into this dir so it is in the docker build context
	cp ../THIRD_PARTY_LICENSES.txt .
	docker build --pull -f Dockerfile \
		--build-arg VERRAZZANO_APPLICATION_OPERATOR_IMAGE="${VERRAZZANO_APPLICATION_OPERATOR_IMAGE}" \
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
	$(GO) test -v  ./internal/... ./controllers/... ./apis/...

.PHONY: coverage
coverage: unit-test
	./build/scripts/coverage.sh html

#
# Test-related tasks
#
# On Jenkins we have pre-created clusters ready to go that are named specifically for the integ-test case
# Not conditionalizing this here so the local and Jenkins builds will use the same cluster name, but we could
# keep the local stuff the same as it was
CLUSTER_NAME = vpo-integ
VERRAZZANO_NS = verrazzano-install
BUILD-DEPLOY = build/deploy
DEPLOY = deploy
OPERATOR_SETUP = test/operatorsetup

ifdef JENKINS_URL
	CLUSTER_DUMP_LOCATION = ${WORKSPACE}/platform-operator-integ-cluster-dump
else
	CLUSTER_DUMP_LOCATION = platform-operator-integ-cluster-dump
endif

.PHONY: integ-test
integ-test: create-cluster
	echo 'Load docker image for the verrazzano-platform-operator...'

	echo 'Deploy verrazzano platform operator ...'
ifdef JENKINS_URL
	kind load docker-image --name ${CLUSTER_NAME} ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
	kubectl apply -f ${WORKSPACE}/generated-operator.yaml || (echo 'Platform operator install failed, capturing kind cluster dump'; ../tools/scripts/k8s-dump-cluster.sh -d ${CLUSTER_DUMP_LOCATION}; exit 1)
else
	kind load docker-image --name ${CLUSTER_NAME} ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
	mkdir -p build/deploy
	cat config/deploy/verrazzano-platform-operator.yaml | sed -e "s|IMAGE_NAME|${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}|g" > ${BUILD-DEPLOY}/operator.yaml
	cat config/crd/bases/install.verrazzano.io_verrazzanos.yaml >> ${BUILD-DEPLOY}/operator.yaml
	cat config/crd/bases/clusters.verrazzano.io_verrazzanomanagedclusters.yaml >> ${BUILD-DEPLOY}/operator.yaml
	kubectl apply -f ${BUILD-DEPLOY}/operator.yaml || (echo 'Platform operator install failed, capturing kind cluster dump'; ../tools/scripts/k8s-dump-cluster.sh -d ${CLUSTER_DUMP_LOCATION}; exit 1)
endif
	echo 'Run tests...'
	ginkgo -v --keepGoing -cover test/integ/... || (echo 'Platform operator tests failed, capturing kind cluster dump'; ../tools/scripts/k8s-dump-cluster.sh -d ${CLUSTER_DUMP_LOCATION}; exit 1)

.PHONY: create-cluster
create-cluster:
ifdef JENKINS_URL
	echo "Jenkins has pre-created vpo-integ cluster ready to use"
	# Get the ip address of the container running the kube apiserver
	# and update the kubeconfig file to point to that address, instead of localhost
	sed -i -e "s|127.0.0.1.*|`docker inspect ${CLUSTER_NAME}-control-plane | jq '.[].NetworkSettings.Networks[].IPAddress' | sed 's/"//g'`:6443|g" ${KUBECONFIG}
	cat ${KUBECONFIG} | grep server

	$$(X=$$(docker inspect $$(docker ps | grep "jenkins-runner" | awk '{ print $$1 }') | jq '.[].NetworkSettings.Networks' | grep -q kind ; echo $$?); if [[ ! $$X -eq "0" ]]; then docker network connect kind $$(docker ps | grep "jenkins-runner" | awk '{ print $$1 }'); fi)
else
	echo 'Create cluster...'
	echo 'KUBECONFIG=${KUBECONFIG}'
	HTTP_PROXY="" HTTPS_PROXY="" http_proxy="" https_proxy="" time kind create cluster -v 1 \
		--name ${CLUSTER_NAME} \
		--wait 5m \
		--config=test/kind-config.yaml
	kubectl config set-context kind-${CLUSTER_NAME}
endif

.PHONY: delete-cluster
delete-cluster:
ifdef JENKINS_URL
	echo "Don't bother deleting the cluster on Jenkins, it will not be re-used by anything else"
else
	kind delete cluster --name ${CLUSTER_NAME}
endif

.PHONY: push-tag
push-tag:
	PUBLISH_TAG="${DOCKER_IMAGE_TAG}"; \
	echo "Tagging and pushing image ${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	docker pull "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}"; \
	docker tag "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}" "${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	docker push "${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"

.PHONY: create-test-deploy
create-test-deploy:
	if [ -n "${VZ_DEV_IMAGE}" ]; then \
		echo "Building local operator deployment resource file in /tmp/operator.yaml, VZ_DEV_IMAGE=${VZ_DEV_IMAGE}"; \
		cat config/deploy/verrazzano-platform-operator.yaml | sed -e "s|IMAGE_NAME|${VZ_DEV_IMAGE}|g" > /tmp/operator.yaml; \
		cat config/crd/bases/install.verrazzano.io_verrazzanos.yaml >> /tmp/operator.yaml; \
		cat config/crd/bases/clusters.verrazzano.io_verrazzanomanagedclusters.yaml >> /tmp/operator.yaml; \
	else \
		echo "VZ_DEV_IMAGE not defined, please set it to a valid image name/tag"; \
	fi
