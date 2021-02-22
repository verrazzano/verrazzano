# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

NAME:=verrazzano-application-operator
REPO_NAME:=verrazzano-application-operator


TIMESTAMP := $(shell date -u +%Y%m%d%H%M%S)
DOCKER_IMAGE_NAME ?= ${NAME}-dev
DOCKER_IMAGE_TAG ?= local-${TIMESTAMP}-$(shell git rev-parse --short HEAD)

CONTROLLER_GEN_VERSION ?= $(shell go list -m -f '{{.Version}}' sigs.k8s.io/controller-tools)
CREATE_LATEST_TAG=0

CRD_OPTIONS ?= "crd:crdVersions=v1"

ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),docker-push push-tag))
	ifndef DOCKER_REPO
		$(error DOCKER_REPO must be defined as the name of the Docker repository where image will be pushed)
	endif
	ifndef DOCKER_NAMESPACE
		$(error DOCKER_NAMESPACE must be defined as the name of the Docker namespace where image will be pushed)
	endif
	DOCKER_IMAGE_FULLNAME = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${DOCKER_IMAGE_NAME}
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
ENV_NAME=verrazzano-application-operator
GO ?= CGO_ENABLED=0 GO111MODULE=on GOPRIVATE=github.com/verrazzano go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}
CRD_PATH=config/crd/bases

ifdef JENKINS_URL
	CLUSTER_DUMP_LOCATION = ${WORKSPACE}/application-operator-integ-cluster-dump
else
	CLUSTER_DUMP_LOCATION = application-operator-integ-cluster-dump
endif

.PHONY: build
build: go-fmt go-vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run:
	# To enable webhooks: $(GO) run main.go --kubeconfig=${KUBECONFIG} --cert-dir=build/webhook-certs
	$(GO) run main.go --kubeconfig=${KUBECONFIG} --enable-webhooks=false --metrics-addr=localhost:0

# Install CRDs into a cluster
.PHONY: install-crds
install-crds: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
.PHONY: uninstall-crds
uninstall-crds: manifests
	kustomize build config/crd | kubectl delete -f -

.PHONY: check
check: go-fmt go-vet go-ineffassign go-lint

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests: go-mod controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role paths="./..." output:crd:artifacts:config=config/crd/bases
	# Add copyright headers to the kubebuilder generated CRDs
	./hack/add-crd-header.sh
	# Add copyright headers to the kubebuilder generated manifests
	./hack/add-yml-header.sh PROJECT
	./hack/add-yml-header.sh config/rbac/role.yaml

# Generate code
.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# find or download controller-gen
# download controller-gen if necessary
.PHONY: controller-gen
controller-gen:
ifeq (, $(shell command -v controller-gen))
	$(GO) get sigs.k8s.io/controller-tools/cmd/controller-gen
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

# Generate mocks
.PHONY: mock-gen
mock-gen:
	mockgen -destination=mocks/controller_client_mock.go -package=mocks -copyright_file=hack/boilerplate.go.txt sigs.k8s.io/controller-runtime/pkg/client Client,StatusWriter
	mockgen -destination=mocks/controller_manager_mock.go -package=mocks -copyright_file=hack/boilerplate.go.txt sigs.k8s.io/controller-runtime Manager
	mockgen -destination=mocks/logging_scope_handler_mock.go -package=mocks -copyright_file=hack/boilerplate.go.txt github.com/verrazzano/verrazzano/application-operator/controllers/loggingscope Handler
	mockgen -destination=controllers/loggingscope/fluentd_mock.go -package=loggingscope -copyright_file=hack/boilerplate.go.txt github.com/verrazzano/verrazzano/application-operator/controllers/loggingscope FluentdManager

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

.PHONY: go-fmt
go-fmt:
	gofmt -s -e -d $(shell find . -name "*.go" | grep -v /vendor/ | grep -v /pkg/assets/) > error.txt
	if [ -s error.txt ]; then\
		cat error.txt;\
		rm error.txt;\
		exit 1;\
	fi
	rm error.txt

.PHONY: go-vet
go-vet:
	$(GO) vet $(shell go list ./... | grep -v github.com/verrazzano/verrazzano-application-operator/pkg/assets)

.PHONY: go-lint
go-lint:
	@{ \
	set -eu ; \
	GOLINT_VERSION=$$(go list -m -f '{{.Version}}' golang.org/x/lint) ; \
	${GO} get golang.org/x/lint/golint@$${GOLINT_VERSION} ; \
	}
	golint -set_exit_status $(shell go list ./... | grep -v github.com/verrazzano/verrazzano-application-operator/pkg/assets)

.PHONY: go-ineffassign
go-ineffassign:
	@{ \
	set -eu ; \
	INEFFASSIGN_VERSION=$$(go list -m -f '{{.Version}}' github.com/gordonklaus/ineffassign) ; \
	${GO} get github.com/gordonklaus/ineffassign@$${INEFFASSIGN_VERSION} ; \
	}
	ineffassign $(shell go list ./...)

.PHONY: go-mod
go-mod:
	$(GO) mod tidy
	$(GO) mod vendor

#
# Docker-related tasks
#
.PHONY: docker-clean
docker-clean:
	rm -rf ${DIST_DIR}

.PHONY: docker-build
docker-build: go-build-linux
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
	$(GO) test -v ./apis/... ./controllers/... ./internal/...

.PHONY: coverage
coverage: unit-test
	./build/scripts/coverage.sh html

#
# Test-related tasks
#
CLUSTER_NAME = verrazzano
VERRAZZANO_NS = verrazzano-system
DEPLOY = deploy
OPERATOR_SETUP = test/operatorsetup
OAM_RUNTIME_IMAGE="ghcr.io/verrazzano/oam-kubernetes-runtime:v0.3.0-20210222205541-9e8d4fb"

# These exports are needed for the install.sh
export KUBECONFIG=${HOME}/.kube/config
export VERRAZZANO_KUBECONFIG=${HOME}/.kube/config
export VERRAZZANO_APP_OP_IMAGE=${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

.PHONY: integ-test
integ-test: create-cluster
	echo 'Load Docker image for the OAM runtime...'
	docker pull ${OAM_RUNTIME_IMAGE}
	kind load docker-image --name ${CLUSTER_NAME} ${OAM_RUNTIME_IMAGE}

	echo 'Load Docker image for the Verrazzano application operator...'
	kind load docker-image --name ${CLUSTER_NAME} ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

	echo 'Install OAM runtime and Verrazzano application operator...'
	installer/install.sh || (echo 'Application operator install failed, capturing kind cluster dump'; ../tools/scripts/k8s-dump-cluster.sh -d ${CLUSTER_DUMP_LOCATION}; exit 1)

	echo 'Run tests...'
	ginkgo -v --keepGoing -cover test/integ/... || (echo 'Application operator tests failed, capturing kind cluster dump'; ../tools/scripts/k8s-dump-cluster.sh -d ${CLUSTER_DUMP_LOCATION}; exit 1)

.PHONY: create-cluster
create-cluster:
ifdef JENKINS_URL
	./build/scripts/cleanup.sh ${CLUSTER_NAME}
endif
	echo 'Create cluster...'
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
