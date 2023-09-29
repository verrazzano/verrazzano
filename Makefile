# Copyright (c) 2020, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

.DEFAULT_GOAL := help
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

include make/quality.mk

SCRIPT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/build
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TOOLS_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/tools

ifneq "$(MAKECMDGOALS)" ""
ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),docker-push create-test-deploy))
ifndef DOCKER_REPO
    $(error DOCKER_REPO must be defined as the name of the docker repository where image will be pushed)
endif
ifndef DOCKER_NAMESPACE
    $(error DOCKER_NAMESPACE must be defined as the name of the docker namespace where image will be pushed)
endif
endif
endif

TIMESTAMP := $(shell date -u +%Y%m%d%H%M%S)
DOCKER_IMAGE_TAG ?= local-${TIMESTAMP}-$(shell git rev-parse --short HEAD)
VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME ?= verrazzano-platform-operator-dev
VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME ?= verrazzano-application-operator-dev
VERRAZZANO_CLUSTER_OPERATOR_IMAGE_NAME ?= verrazzano-cluster-operator-dev
VERRAZZANO_AUTHPROXY_IMAGE_NAME ?= verrazzano-authproxy-dev

VERRAZZANO_PLATFORM_OPERATOR_IMAGE = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
VERRAZZANO_APPLICATION_OPERATOR_IMAGE = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
VERRAZZANO_CLUSTER_OPERATOR_IMAGE = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${VERRAZZANO_CLUSTER_OPERATOR_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
VERRAZZANO_AUTHPROXY_IMAGE = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${VERRAZZANO_AUTHPROXY_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

CURRENT_YEAR = $(shell date +"%Y")

PARENT_BRANCH ?= origin/master

GO ?= CGO_ENABLED=0 GO111MODULE=on GOPRIVATE=github.com/verrazzano go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}

.PHONY: clean
clean: ## remove coverage and test results
	find . -name coverage.cov -exec rm {} \;
	find . -name coverage.html -exec rm {} \;
	find . -name coverage.raw.cov -exec rm {} \;
	find . -name \*-test-result.xml -exec rm {} \;
	find . -name coverage.xml -exec rm {} \;
	find . -name unit-test-coverage-number.txt -exec rm {} \;

##@ Build

.PHONY: docker-push
docker-push: ## build and push all images
	(cd authproxy; make docker-push DOCKER_IMAGE_NAME=${VERRAZZANO_AUTHPROXY_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG})
	(cd cluster-operator; make docker-push DOCKER_IMAGE_NAME=${VERRAZZANO_CLUSTER_OPERATOR_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG})
	(cd application-operator; make docker-push DOCKER_IMAGE_NAME=${VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG})
	(cd platform-operator; make docker-push DOCKER_IMAGE_NAME=${VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG} VERRAZZANO_APPLICATION_OPERATOR_IMAGE=${VERRAZZANO_APPLICATION_OPERATOR_IMAGE} VERRAZZANO_CLUSTER_OPERATOR_IMAGE=${VERRAZZANO_CLUSTER_OPERATOR_IMAGE} VERRAZZANO_AUTHPROXY_IMAGE=${VERRAZZANO_AUTHPROXY_IMAGE})

.PHONY: docker-push-debug
docker-push-debug: ## build and push all images
	(cd authproxy; make docker-push-debug DOCKER_IMAGE_NAME=${VERRAZZANO_AUTHPROXY_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG})
	(cd cluster-operator; make docker-push-debug DOCKER_IMAGE_NAME=${VERRAZZANO_CLUSTER_OPERATOR_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG})
	(cd application-operator; make docker-push-debug DOCKER_IMAGE_NAME=${VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG})
	(cd platform-operator; make docker-push-debug DOCKER_IMAGE_NAME=${VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG} VERRAZZANO_APPLICATION_OPERATOR_IMAGE=${VERRAZZANO_APPLICATION_OPERATOR_IMAGE} VERRAZZANO_CLUSTER_OPERATOR_IMAGE=${VERRAZZANO_CLUSTER_OPERATOR_IMAGE} VERRAZZANO_AUTHPROXY_IMAGE=${VERRAZZANO_AUTHPROXY_IMAGE})

.PHONY: create-test-deploy
create-test-deploy: docker-push create-test-deploy-common ## build and push all images, then create operator.yaml file for Verrazzano deployment

.PHONY: create-test-deploy-debug
create-test-deploy-debug: docker-push-debug create-test-deploy-common ## build and push all images, then create operator.yaml file for Verrazzano deployment (debug)

.PHONY: create-test-deploy-common
create-test-deploy-common: ## create operator.yaml file for Verrazzano deployment
	(cd platform-operator; make create-test-deploy VZ_DEV_IMAGE=${VERRAZZANO_PLATFORM_OPERATOR_IMAGE} VZ_APP_OP_IMAGE=${VERRAZZANO_APPLICATION_OPERATOR_IMAGE} VZ_CLUSTER_OP_IMAGE=${VERRAZZANO_CLUSTER_OPERATOR_IMAGE} VZ_AUTHPROXY_IMAGE=${VERRAZZANO_AUTHPROXY_IMAGE})

.PHONY: test-platform-operator-install
test-platform-operator-install: ## install VPO from operator.yaml
	kubectl apply -f platform-operator/build/deploy/operator.yaml
	kubectl -n verrazzano-install rollout status deployment/verrazzano-platform-operator

.PHONY: test-platform-operator-remove
test-platform-operator-remove: ## delete VPO from operator.yaml
	kubectl delete -f platform-operator/build/deploy/operator.yaml

.PHONY: test-platform-operator-install-logs
test-platform-operator-install-logs: ## tail VPO logs
	kubectl logs -f -n verrazzano-install $(shell kubectl get pods -n verrazzano-install --no-headers | grep "^verrazzano-platform-operator-" | cut -d ' ' -f 1)

##@ Testing

.PHONY: precommit
precommit: precommit-check precommit-build unit-test-coverage ## run all precommit checks

.PHONY: precommit-nocover
precommit-nocover: precommit-check precommit-build unit-test ## run precommit checks without code coverage check

.PHONY: precommit-check
precommit-check: check-tidy check check-tests copyright-check check-bom-tags ## run precommit checks without unit testing

.PHONY: precommit-build
precommit-build:  ## go build the project
	go build ./...

unit-test-coverage: export COVERAGE_EXCLUSIONS ?= tests/e2e|tools/psr|tools/charts-manager/vcm|platform-operator/experimental|platform-operator/controllers/module|platform-operator/controllers/verrazzano/controller
.PHONY: unit-test-coverage
unit-test-coverage:  ## run unit tests with coverage
	${SCRIPT_DIR}/coverage.sh html

.PHONY: unit-test-coverage-ratcheting
unit-test-coverage-ratcheting:  ## run unit tests with coverage ratcheting
	${SCRIPT_DIR}/coverage-number-comparison.sh

.PHONY: unit-test
unit-test:  ## run all unit tests in project
	go test $$(go list ./... | grep -Ev "/tests/e2e|/tools/psr|tools/charts-manager/vcm")

#
#  Compliance check targets
#

##@ Compliance

.PHONY: copyright-test
copyright-test: ## run the tests for the copyright checker
	(cd tools/copyright; go test .)

.PHONY: copyright-check-year
copyright-check-year: copyright-test ## check copyright notices have correct current year
	go run tools/copyright/copyright.go --enforce-current $(shell git log --since=01-01-${CURRENT_YEAR} --name-only --oneline --pretty="format:" | sort -u)

.PHONY: copyright-check
copyright-check: copyright-test copyright-check-year  ## check copyright notices are correct
	go run tools/copyright/copyright.go .

.PHONY: copyright-check-local
copyright-check-local: copyright-test  ## check copyright notices are correct in local working copy
	go run tools/copyright/copyright.go --verbose --enforce-current  $(shell git status --short | cut -c 4-)

.PHONY: copyright-check-branch
copyright-check-branch: copyright-check ## check copyright notices are correct in parent branch
	go run tools/copyright/copyright.go --verbose --enforce-current $(shell git diff --name-only ${PARENT_BRANCH})

.PHONY: check-bom-tags
check-bom-tags: ## check for any image tag problems in the BOM
	release/scripts/check_image_tags.sh platform-operator/verrazzano-bom.json

#
# Quality checks on acceptance tests
#

##@ Quality

.PHONY: check-tests
check-tests: check-eventually ## check test code for known quality issues

.PHONY: check-eventually
check-eventually: check-eventually-test ## check for correct use of Gomega Eventually func
	go run tools/eventually-checker/check_eventually.go tests/e2e

.PHONY: check-eventually-test
check-eventually-test: ## run tests for Gomega Eventually checker
	(cd tools/eventually-checker; go test .)

.PHONY: check-tidy
check-tidy: ## check if go mod tidy results in no changes
	go mod tidy
	ci/scripts/check_if_clean_after_generate.sh
