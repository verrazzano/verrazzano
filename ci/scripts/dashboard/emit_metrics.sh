#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ -z "$SAURON_CRED" ] || [ -z "$GIT_COMMIT" ] || [ -z "$JOB_NAME" ] || [ -z "$BUILD_NUMBER" ] || [ -z "$BRANCH_NAME" ] || [ -z "$PROMETHEUS_GW_URL" ] ; then
  echo "One or more required environment variables are not set."
  exit 1
fi

if [ -z $1 ]; then
    echo "The directory containing test report is required."
    exit 1
fi
TEST_REPORT_DIR=$1

if [ -z $2 ]; then
    echo "The test environment is required."
    exit 1
fi
TEST_ENV=$2

cd ${SCRIPT_DIR}/main
GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go --report-dir="${TEST_REPORT_DIR}" --prometheus-credential="${SAURON_CRED}" --prometheus-url="${PROMETHEUS_GW_URL}" --commit-sha="${GIT_COMMIT}" --test-env="${TEST_ENV}" --branch-name="${BRANCH_NAME}" --build-number="${BUILD_NUMBER}" --job-name="${JOB_NAME}"

