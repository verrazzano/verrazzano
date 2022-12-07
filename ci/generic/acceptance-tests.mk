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

.PHONY: verify-all
verify-all: verify-infra-all verify-deployment-all

.PHONY: verify-infra-all
verify-infra-all: verify-install verify-scripts verify-infra verify-security-rbac verify-system-metrics verify-console

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

.PHONY: verify-deployment-all
verify-deployment-all: verify-opensearch-topology verify-istio-authz verify-deployment-workload-metrics \
	verify-system-logging verify-opensearch-logging verify-helidon-logging verify-helidon-metrics \
	verify-examples-helidon verify-workloads verify-console-ingress verify-wls-loggingtraits verify-poko-metricsbinding \
	verify-security-netpol

.PHONY: verify-opensearch-topology
verify-opensearch-topology:
	TEST_SUITES=opensearch/topology/... make test

.PHONY: verify-istio-authz
verify-istio-authz:
	TEST_SUITES=istio/authz/... make run-sequential

.PHONY: verify-deployment-workload-metrics
verify-deployment-workload-metrics:
	TEST_SUITES=metrics/deploymetrics/... make run-sequential

.PHONY: verify-system-logging
verify-system-logging:
	TEST_SUITES=logging/system/... make run-sequential

.PHONY: verify-opensearch-logging
verify-opensearch-logging:
	TEST_SUITES=logging/opensearch/... make run-sequential

.PHONY: verify-helidon-logging
verify-helidon-logging:
	TEST_SUITES=logging/helidon/... make run-sequential

.PHONY: verify-helidon-metrics
verify-helidon-metrics:
	TEST_SUITES=examples/helidonmetrics/... make run-sequential

.PHONY: verify-examples-helidon
verify-examples-helidon:
	TEST_SUITES=examples/helidon/... make test

.PHONY: verify-workloads
verify-workloads:
	TEST_SUITES=workloads/... make run-sequential

.PHONY: verify-console-ingress
verify-console-ingress:
	TEST_SUITES=ingress/console/... make run-sequential

.PHONY: verify-wls-loggingtraits
verify-wls-loggingtraits:
	TEST_SUITES=loggingtrait/... make run-sequential

.PHONY: verify-poko-metricsbinding
verify-poko-metricsbinding:
	TEST_SUITES=metricsbinding/... make run-sequential

.PHONY: verify-security-netpol
verify-security-netpol:
	TEST_SUITES=security/netpol/... make run-sequential

.PHONY: dumplogs
dumplogs:
	${CI_SCRIPTS_DIR}/dumpRunLogs.sh ${DUMP_ROOT_DIRECTORY}

test-reports: export TEST_REPORT ?= "test-report.xml"
test-reports: export TEST_REPORT_DIR ?= "${WORKSPACE}/tests/e2e"
.PHONY: test-reports
test-reports:
	# Copy the generated test reports to WORKSPACE to archive them
	mkdir -p ${TEST_REPORT_DIR}
	cd ${GO_REPO_PATH}/verrazzano/tests/e2e
	find . -name "${TEST_REPORT}" | cpio -pdm ${TEST_REPORT_DIR}

.PHONY: pipeline-artifacts
pipeline-artifacts: dumplogs test-reports

