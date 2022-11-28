# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ../make/kind.mk
include ../make/install.mk
include ../make/cleanup.mk

.PHONY: all
all: setup install test

setup: export K8S_VERSION = 1.24
.PHONY: setup
setup:
	@echo "Running test setup"
	make setup-kind

# Peforms an install of Verrazzano to the target system
.PHONY: install
install: 
	@echo "Installing Verrazzano"
	make install-verrazzano

# Executes test suite(s) against the target Verrazzano install
test: export TEST_SUITES ?= verify-install/...
test: export PSR_COMMAND ?= psrctl
test: export TEST_NAMESPACE ?= psrtest
.PHONY: test
test:
	@echo "Running tests ${TEST_SUITES}"
	kubectl create ns ${TEST_NAMESPACE} || true
	kubectl label ns --overwrite=true ${TEST_NAMESPACE} verrazzano-managed=true istio-injection=enabled
	${TEST_SCRIPTS_DIR}/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}" "${TEST_NAMESPACE}"
	${PSR_COMMAND} start -s ops-s2 -n psrtest
	sleep 120
	${PSR_COMMAND} stop -s ops-s2 -n psrtest
	@echo "Testing complete"

# Executes an upgrade to a new Verrazzano version from the initially installed version
.PHONY: cleanup
cleanup: pipeline-artifacts clean-kind
	@echo "Running test cleanup"
