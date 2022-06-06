#!/usr/bin/env bash
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
if [ ! -z "${kubeConfig}" ]; then
    export KUBECONFIG="${kubeConfig}"
fi
if [ -z "${TEST_SUITES}" ]; then
  echo "${0}: No test suites specified"
  exit 0
fi

TEST_DUMP_ROOT=${TEST_DUMP_ROOT:-"."}
SEQUENTIAL_SUITES=${SEQUENTIAL_SUITES:-false}
KUBECONFIG_ROOT=${KUBECONFIG_ROOT:-${WORKSPACE}/kubeconfigs}

GINGKO_ARGS=${GINGKO_ARGS:-"-v --keep-going --no-color"}
if [ "${RUN_PARALLEL}" == "true" ]; then
  GINGKO_ARGS="${GINGKO_ARGS} -p"
fi
if [ "${RANDOMIZE_TESTS}" == "true" ]; then
  GINGKO_ARGS="${GINGKO_ARGS} --randomize-all"
fi
if [ -n "${TAGGED_TESTS}" ]; then
  GINGKO_ARGS="${GINGKO_ARGS} -tags=${TAGGED_TESTS}"
fi
if [ -n "${INCLUDED_TESTS}" ]; then
  GINGKO_ARGS="${GINGKO_ARGS} --focus-file=${INCLUDED_TESTS}"
fi
if [ -n "${EXCLUDED_TESTS}" ]; then
  GINGKO_ARGS="${GINGKO_ARGS} --skip-file=${EXCLUDED_TESTS}"
fi
if [ -n "${DRY_RUN}" ]; then
  GINGKO_ARGS="${GINGKO_ARGS} --dry-run"
fi
if [ -n "${SKIP_DEPLOY}" ]; then
  TEST_ARGS="${TEST_ARGS} --skip-deploy=${SKIP_DEPLOY}"
fi
if [ -n "${SKIP_UNDEPLOY}" ]; then
  TEST_ARGS="${TEST_ARGS} --skip-undeploy=${SKIP_UNDEPLOY}"
fi
set -x

if [ -n "${TEST_ARGS}" ]; then
  TEST_ARGS="-- ${TEST_ARGS}"
fi

CLUSTER_COUNT=${CLUSTER_COUNT:-1}
count=1
while [ ${count}  -le ${CLUSTER_COUNT} ]; do
  echo "Using KUBECONFIG location ${KUBECONFIG_DIR}/$count"
  mkdir -p ${KUBECONFIG_ROOT}/$count
  export KUBECONFIG=${KUBECONFIG_ROOT}/$count/kube_config
  export DUMP_KUBECONFIG="${KUBECONFIG}"
  export TEST_REPORT_DIR="${WORKSPACE}/tests/${count}"
  export TEST_REPORT_ARGS="--output-dir ${TEST_REPORT_DIR}"

  cd ${TEST_ROOT}
  mkdir -p "${TEST_REPORT_DIR}"
  ginkgo ${GINGKO_ARGS} ${TEST_REPORT_ARGS} ${TEST_SUITES} ${TEST_ARGS}

  let count=count+1
done
