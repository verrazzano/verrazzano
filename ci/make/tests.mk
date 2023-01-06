# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include env.mk

export TEST_ROOT ?= ${VZ_ROOT}/tests/e2e
export DUMP_ROOT_DIRECTORY ?= ${WORKSPACE}/cluster-snapshots
export GINGKO_ARGS ?= -v --keep-going --no-color --junit-report=test-report.xml --keep-separate-reports=true

run-test-parallel: export RANDOMIZE_TESTS = false
run-test-parallel: export RUN_PARALLEL = true
.PHONY: run-test-parallel
run-test-parallel: run-test

run-test-sequential: export RANDOMIZE_TESTS = false
run-test-sequential: export RUN_PARALLEL = false
.PHONY: run-sequential
run-test-sequential: run-test

run-test-randomize: export RANDOMIZE_TESTS = true
run-test-randomize: export RUN_PARALLEL = true
.PHONY: run-test-randomize
run-test-randomize: run-test

.PHONY: run-test
run-test: export TEST_REPORT ?= test-report.xml
run-test: export TEST_REPORT_DIR ?= ${WORKSPACE}/tests/e2e
run-test:
	${CI_SCRIPTS_DIR}/run-ginkgo.sh

.PHONY: dumplogs
dumplogs:
	${CI_SCRIPTS_DIR}/dumpRunLogs.sh ${DUMP_ROOT_DIRECTORY}

test-reports: export TEST_REPORT ?= test-report.xml
test-reports: export TEST_REPORT_DIR ?= ${WORKSPACE}/tests/e2e
.PHONY: test-reports
test-reports:
	# Copy the generated test reports to WORKSPACE to archive them
	# - Seems to not be working, handling this now in the run-ginkgo.sh script
	mkdir -p ${TEST_REPORT_DIR}
	rsync -v -a --prune-empty-dirs --exclude '*.go'  --include '${TEST_REPORT}' ${TEST_ROOT} ${TEST_REPORT_DIR}

.PHONY: pipeline-artifacts
pipeline-artifacts: dumplogs test-reports
