# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include global-env.mk

export DUMP_ROOT_DIRECTORY ?= ${WORKSPACE}/cluster-dumps
export GINGKO_ARGS ?= -v --keep-going --no-color

console-test: export DUMP_DIRECTORY ?= ${DUMP_ROOT_DIRECTORY}/console
PHONY: console-test
console-test:
	${CI_SCRIPTS_DIR}/run_console_tests.sh

run-test: export RANDOMIZE_TESTS ?= true
run-test: export RUN_PARALLEL ?= true
.PHONY: run-test
run-test:
	${CI_SCRIPTS_DIR}/run-ginkgo.sh
