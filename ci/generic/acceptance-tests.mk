# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include global-env.mk

export TEST_DUMP_ROOT ?= ${WORKSPACE}/cluster-dumps

#verify-install: export DUMP_DIRECTORY ?= ${TEST_DUMP_ROOT}/verify-install
#verify-install: export TEST_SUITES := verify-install
#.PHONY: verify-install
#verify-install: run-test

run-test: export RANDOMIZE_TESTS ?= true
run-test: export RUN_PARALLEL ?= true
run-test: export SEQUENTIAL_SUITES ?= false
.PHONY: run-test
run-test:
	${CI_SCRIPTS_DIR}/run-ginkgo.sh
