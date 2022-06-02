# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include global-env.mk

export DUMP_ROOT_DIRECTORY ?= ${WORKSPACE}/cluster-dumps
export GINGKO_ARGS ?= -v --keep-going --no-color

run-test: export RANDOMIZE_TESTS ?= true
run-test: export RUN_PARALLEL ?= true
.PHONY: run-test
run-test:
	${CI_SCRIPTS_DIR}/run-ginkgo.sh

run-test: export RANDOMIZE_TESTS := false
run-test: export RUN_PARALLEL := false
.PHONY: run-sequential
run-sequential: run-test

.PHONY: verify-install
verify-install:
	TEST_SUITES=verify-install/... make test

.PHONY: verify-scripts
verify-scripts:
	TEST_SUITES=scripts/... make test

.PHONY: verify-infra
verify-infra:
	TEST_SUITES=verify-infra/... make test

.PHONY: verify-security-rbac
verify-security-rbac:
	TEST_SUITES=security/rbac/... make run-sequential

.PHONY: verify-system-metrics
verify-system-metrics:
	TEST_SUITES=metrics/syscomponents/... make run-sequential

verify-console: export DUMP_DIRECTORY ?= ${DUMP_ROOT_DIRECTORY}/console
PHONY: verify-console
verify-console:
	${CI_SCRIPTS_DIR}/run_console_tests.sh

