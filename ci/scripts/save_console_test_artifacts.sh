#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
if [ -z "${GO_REPO_PATH}" ] || [ -z "${WORKSPACE}" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

cd ${GO_REPO_PATH}/console
# Copy artifacts to workspace, ignore if copy fails
mkdir -p ${WORKSPACE}/console/screenshots
mkdir -p ${WORKSPACE}/console/logs
ls Screenshot*.png || true
cp Screenshot*.png ${WORKSPACE}/console/screenshots || true
ls ConsoleLog*.log || true
cp ConsoleLog*.log ${WORKSPACE}/console/logs || true
