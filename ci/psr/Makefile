# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ../make/kind.mk
include ../make/install.mk
include ../make/cleanup.mk
include ../make/tests.mk

export PSR_PATH ?= ${VZ_ROOT}/tools/psr

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
test: export TEST_ROOT = ${PSR_PATH}/tests
test: export TEST_SUITES ?= ${TEST_ROOT}/...
test: export PSR_COMMAND ?= psrctl
test: export TEST_NAMESPACE ?= psrtest
test: export IMAGE_PULL_SECRET ?= verrazzano-container-registry
test: export DOCKER_REGISTRY ?= ghcr.io
.PHONY: test
test: run-test-sequential

# Executes an upgrade to a new Verrazzano version from the initially installed version
.PHONY: cleanup
cleanup: pipeline-artifacts clean-kind
	@echo "Running test cleanup"
