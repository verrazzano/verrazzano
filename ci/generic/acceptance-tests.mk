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
kind-acceptance-tests: setup install
	${RUNGINKGO} verify-install/... verify-infra/... scripts/...
	RUN_PARALLEL=false 	${RUNGINKGO} security/rbac/...  metrics/syscomponents/...
	${CI_SCRIPTS_DIR}/run_console_tests.sh	${RUNGINKGO} opensearch/topology/... examples/helidon/...
	RUN_PARALLEL=false 	${RUNGINKGO} istio/authz/... metrics/deploymetrics/... logging/system/... logging/opensearch/...  logging/helidon/... examples/helidonmetrics/... workloads/... ingress/console/... loggingtrait/... metricsbinding/... security/netpol/...

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

.PHONY: jobmetrics
jobmetrics:
	${RUNGINKGO} jobmetrics/...

.PHONY: pipeline-artifacts
pipeline-artifacts: dumplogs test-reports

# Executes an upgrade to a new Verrazzano version from the initially installed version
.PHONY: cleanup
cleanup: pipeline-artifacts clean-kind

.PHONY: verify-all
verify-all: verify-infra-all verify-deployment-all

.PHONY: verify-infra-all
verify-infra-all: verify-install verify-scripts verify-infra verify-security-rbac verify-system-metrics verify-console

.PHONY: verify-install
verify-install:
	${RUNGINKGO} verify-install/...

.PHONY: jobmetrics
jobmetrics:
	${RUNGINKGO} jobmetrics/...

.PHONY: verify-scripts
verify-scripts:
	${RUNGINKGO} scripts/...

.PHONY: verify-infra
verify-infra:
	${RUNGINKGO} verify-infra/...

.PHONY: verify-security-rbac
verify-security-rbac:
	RUN_PARALLEL=false ${RUNGINKGO} security/rbac/...

.PHONY: verify-system-metrics
verify-system-metrics:
	RUN_PARALLEL=false ${RUNGINKGO} metrics/syscomponents/...

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
	${RUNGINKGO} opensearch/topology/...

.PHONY: verify-istio-authz
verify-istio-authz:
	RUN_PARALLEL=false ${RUNGINKGO} istio/authz/...

.PHONY: verify-deployment-workload-metrics
verify-deployment-workload-metrics:
	RUN_PARALLEL=false ${RUNGINKGO} metrics/deploymetrics/...

.PHONY: verify-system-logging
verify-system-logging:
	RUN_PARALLEL=false ${RUNGINKGO} logging/system/...

.PHONY: verify-opensearch-logging
verify-opensearch-logging:
	RUN_PARALLEL=false ${RUNGINKGO} logging/opensearch/...

.PHONY: verify-helidon-logging
verify-helidon-logging:
	RUN_PARALLEL=false ${RUNGINKGO} logging/helidon/...

.PHONY: verify-helidon-metrics
verify-helidon-metrics:
	RUN_PARALLEL=false ${RUNGINKGO} examples/helidonmetrics/...

.PHONY: verify-examples-helidon
verify-examples-helidon:
	${RUNGINKGO} examples/helidon/...

.PHONY: verify-workloads
verify-workloads:
	RUN_PARALLEL=false ${RUNGINKGO} workloads/...

.PHONY: verify-console-ingress
verify-console-ingress:
	RUN_PARALLEL=false ${RUNGINKGO} ingress/console/...

.PHONY: verify-wls-loggingtraits
verify-wls-loggingtraits:
	RUN_PARALLEL=false ${RUNGINKGO} loggingtrait/...

.PHONY: verify-poko-metricsbinding
verify-poko-metricsbinding:
	RUN_PARALLEL=false ${RUNGINKGO} metricsbinding/...

.PHONY: verify-security-netpol
verify-security-netpol:
	RUN_PARALLEL=false ${RUNGINKGO} security/netpol/...

