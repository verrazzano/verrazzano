# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

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

# Run against the configured Kubernetes cluster in ~/.kube/config
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

.PHONY: check
check: go-fmt go-vet go-ineffassign go-lint

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

.PHONY: go-fmt
go-fmt:
	gofmt -s -e -d $(shell find . -name "*.go" | grep -v /vendor/) > error.txt
	if [ -s error.txt ]; then\
		cat error.txt;\
		rm error.txt;\
		exit 1;\
	fi
	rm error.txt

.PHONY: go-vet
go-vet:
	$(GO) vet $(shell go list ./...)

.PHONY: go-lint
go-lint:
	$(GO) get -u golang.org/x/lint/golint
	golint -set_exit_status $(shell go list ./...)

.PHONY: go-ineffassign
go-ineffassign:
	$(GO) get -u github.com/gordonklaus/ineffassign
	ineffassign $(shell go list ./...)

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	# Add copyright headers to the kubebuilder generated CRDs
	./hack/add-crd-header.sh
	./hack/update-codegen.sh "verrazzano:v1alpha1" "verrazzano" "boilerplate.go.txt"
	./hack/update-codegen.sh "clusters:v1alpha1" "clusters" "boilerplate-clusters.go.txt"

	# Re-generate operator.yaml using template yaml file
	cat config/deploy/verrazzano-platform-operator.yaml | sed -e "s|IMAGE_NAME|$(shell grep --max-count=1 "image:" deploy/operator.yaml | awk '{ print $$2 }')|g" > deploy/operator.yaml
	cat config/crd/bases/install.verrazzano.io_verrazzanos.yaml >> deploy/operator.yaml
	cat config/crd/bases/clusters.verrazzano.io_verrazzanomanagedclusters.yaml >> deploy/operator.yaml

# Generate code
.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# find or download controller-gen
# download controller-gen if necessary
.PHONY: controller-gen
controller-gen:
ifeq (, $(shell command -v controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@
	$(eval CONTROLLER_GEN=$(GOBIN)/controller-gen)
else
	$(eval CONTROLLER_GEN=$(shell command -v controller-gen))
endif
	@{ \
	set -eu; \
	ACTUAL_CONTROLLER_GEN_VERSION=$$(${CONTROLLER_GEN} --version | awk '{print $$2}') ; \
	if [ "$${ACTUAL_CONTROLLER_GEN_VERSION}" != "${CONTROLLER_GEN_VERSION}" ] ; then \
		echo  "Bad controller-gen version $${ACTUAL_CONTROLLER_GEN_VERSION}, please install ${CONTROLLER_GEN_VERSION}" ; \
	fi ; \
	}

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
CLUSTER_NAME = verrazzano
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
	kubectl apply -f deploy/operator.yaml || (echo 'Platform operator install failed, capturing kind cluster dump'; ../tools/scripts/k8s-dump-cluster.sh -d ${CLUSTER_DUMP_LOCATION}; exit 1)
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
	./build/scripts/cleanup.sh ${CLUSTER_NAME}
endif
	echo 'Create cluster...'
	echo 'KUBECONFIG=${KUBECONFIG}'
	HTTP_PROXY="" HTTPS_PROXY="" http_proxy="" https_proxy="" time kind create cluster -v 1 \
		--name ${CLUSTER_NAME} \
		--wait 5m \
		--config=test/kind-config.yaml
	kubectl config set-context kind-${CLUSTER_NAME}
ifdef JENKINS_URL
	# Get the ip address of the container running the kube apiserver
	# and update the kubeconfig file to point to that address, instead of localhost
	sed -i -e "s|127.0.0.1.*|`docker inspect ${CLUSTER_NAME}-control-plane | jq '.[].NetworkSettings.Networks[].IPAddress' | sed 's/"//g'`:6443|g" ${KUBECONFIG}
	cat ${KUBECONFIG} | grep server

	$$(X=$$(docker inspect $$(docker ps | grep "jenkins-runner" | awk '{ print $$1 }') | jq '.[].NetworkSettings.Networks' | grep -q kind ; echo $$?); if [[ ! $$X -eq "0" ]]; then docker network connect kind $$(docker ps | grep "jenkins-runner" | awk '{ print $$1 }'); fi)
endif

.PHONY: delete-cluster
delete-cluster:
	kind delete cluster --name ${CLUSTER_NAME}

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
