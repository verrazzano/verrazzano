# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include global-env.mk

export DUMP_ROOT_DIRECTORY ?= ${WORKSPACE}/cluster-snapshots
export GINGKO_ARGS ?= -v --keep-going --no-color --junit-report=test-report.xml --keep-separate-reports=true

run-test: export RANDOMIZE_TESTS ?= true
run-test: export RUN_PARALLEL ?= true
run-test: export TEST_REPORT ?= "test-report.xml"
run-test: export TEST_REPORT_DIR ?= "${WORKSPACE}/tests/e2e"
.PHONY: run-test
run-test:
	${CI_SCRIPTS_DIR}/run-ginkgo.sh

run-test: export RANDOMIZE_TESTS := false
run-test: export RUN_PARALLEL := false
.PHONY: run-sequential
run-sequential: run-test

.PHONY: kind-acceptance-tests
kind-acceptance-tests: setup install verify-all

.PHONY: verify-all
verify-all: verify-infra-all verify-deployment-all

.PHONY: verify-infra-all
verify-infra-all:  verify-infra-all-parallel verify-infra-all-sequential verify-console

.PHONY: verify-deployment-all
verify-deployment-all: verify-deployment-parallel verify-deployment-sequential

verify-infra-all-parallel: export TEST_SUITES = verify-install/... verify-infra/... scripts/...
.PHONY: verify-infra-all-parallel
verify-infra-all-parallel: run-test

.PHONY: verify-scripts
verify-scripts:
	TEST_SUITES=scripts/... make test

verify-infra-all-sequential: export TEST_SUITES = security/rbac/...  metrics/syscomponents/...
.PHONY: verify-infra-all-sequential
verify-infra-all-sequential: run-test

PHONY: verify-console
verify-console:
	${CI_SCRIPTS_DIR}/run_console_tests.sh

.PHONY: verify-deployment-parallel
verify-deployment-parallel: export TEST_SUITES = opensearch/topology/... examples/helidon/...
verify-deployment-parallel: run-test

.PHONY: verify-deployment-sequential
verify-deployment-sequential: export TEST_SUITES = istio/authz/... metrics/deploymetrics/... logging/system/... logging/opensearch/...  logging/helidon/... examples/helidonmetrics/... workloads/... ingress/console/... loggingtrait/... metricsbinding/... security/netpol/...
verify-deployment-sequential: run-sequential

.PHONY: dumplogs
dumplogs:
	@echo "Dumping test logs to ${DUMP_ROOT_DIRECTORY}"
	${CI_SCRIPTS_DIR}/dumpRunLogs.sh ${DUMP_ROOT_DIRECTORY}

test-reports: export TEST_REPORT ?= "test-report.xml"
test-reports: export TEST_REPORT_DIR ?= "${WORKSPACE}/tests/e2e"
.PHONY: test-reports
test-reports:
	@echo "Copying test reports to ${TEST_REPORT_DIR}"
	# Copy the generated test reports to WORKSPACE to archive them
	# FIXME: should this be copying from ${WORKSPACE} in the pipelines?
	mkdir -p ${TEST_REPORT_DIR}
	cd ${GO_REPO_PATH}/verrazzano/tests/e2e
	find . -name "${TEST_REPORT}" | cpio -pdm ${TEST_REPORT_DIR}

.PHONY: pipeline-artifacts
pipeline-artifacts: dumplogs test-reports

# Executes an upgrade to a new Verrazzano version from the initially installed version
.PHONY: cleanup
cleanup: pipeline-artifacts clean-kind
