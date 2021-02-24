# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),docker-push create-test-deploy))
ifndef DOCKER_REPO
    $(error DOCKER_REPO must be defined as the name of the docker repository where image will be pushed)
endif
ifndef DOCKER_NAMESPACE
    $(error DOCKER_NAMESPACE must be defined as the name of the docker namespace where image will be pushed)
endif
endif

TIMESTAMP := $(shell date -u +%Y%m%d%H%M%S)
DOCKER_IMAGE_TAG ?= local-${TIMESTAMP}-$(shell git rev-parse --short HEAD)
VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME ?= verrazzano-platform-operator-dev
VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME ?= verrazzano-application-operator-dev

VERRAZZANO_PLATFORM_OPERATOR_IMAGE = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
VERRAZZANO_APPLICATION_OPERATOR_IMAGE = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

CURRENT_YEAR = $(shell date +"%Y")

PARENT_BRANCH ?= origin/master

.PHONY: docker-push
docker-push:
	(cd application-operator; make docker-push DOCKER_IMAGE_NAME=${VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG})
	(cd platform-operator; make docker-push DOCKER_IMAGE_NAME=${VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG} VERRAZZANO_APPLICATION_OPERATOR_IMAGE=${VERRAZZANO_APPLICATION_OPERATOR_IMAGE})

.PHONY: create-test-deploy
create-test-deploy: docker-push
	(cd platform-operator; make create-test-deploy VZ_DEV_IMAGE=${VERRAZZANO_PLATFORM_OPERATOR_IMAGE})

.PHONY: test-platform-operator-install
test-platform-operator-install:
	kubectl apply -f /tmp/operator.yaml
	kubectl -n verrazzano-install rollout status deployment/verrazzano-platform-operator

.PHONY: test-platform-operator-remove
test-platform-operator-remove:
	kubectl delete -f /tmp/operator.yaml

.PHONY: test-platform-operator-install-logs
test-platform-operator-install-logs:
	kubectl logs -f -n default $(shell kubectl get pods -n default --no-headers | grep "^verrazzano-install-" | cut -d ' ' -f 1)

.PHONY: copyright-test
copyright-test:
	(cd tools/copyright; go test .)

.PHONY: copyright-check-year
copyright-check-year: copyright-test
	go run tools/copyright/copyright.go --enforce-current $(shell git log --since=01-01-${CURRENT_YEAR} --name-only --oneline --pretty="format:" | sort -u)

.PHONY: copyright-check
copyright-check: copyright-check-year
	go run tools/copyright/copyright.go --verbose .

.PHONY: copyright-check-local
copyright-check-local: copyright-test
	go run tools/copyright/copyright.go --verbose --enforce-current  $(shell git status --short | cut -c 4-)

.PHONY: copyright-check-branch
copyright-check-branch: copyright-check
	go run tools/copyright/copyright.go --verbose --enforce-current $(shell git diff --name-only ${PARENT_BRANCH})
