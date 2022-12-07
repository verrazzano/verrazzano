# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
include global-env.mk

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
