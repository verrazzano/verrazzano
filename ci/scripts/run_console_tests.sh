#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Track starting branch; default to master for Jenkins runs
CONSOLE_BRANCH=master

if [ -z "$GO_REPO_PATH" ]; then
  echo "This script requires GO_REPO_PATH to be set, and is expected to only be called from Jenkins"
  exit 1
fi

cleanupOnError() {
  local rc=$1
  if [ "${rc}" != "0" ]; then
    sh "${GO_REPO_PATH}/verrazzano/ci/scripts/save_console_test_artifacts.sh"
    git checkout ${CONSOLE_BRANCH}
    exit ${rc}
  fi
}
set -e

# Temporarily clone the console repo until it is moved to the Verrazzano repo
cd ${GO_REPO_PATH}
console_sha=${CONSOLE_REPO_BRANCH}
# if no branch override was supplied, use the console SHA present in the Verrazzano BOM
if [[ -z $console_sha ]]; then
  # Get tag of console from BOM
  # shellcheck disable=SC2002
  console_tag=$(cat "verrazzano/platform-operator/verrazzano-bom.json" |jq -r '.components[] | select(.name == "verrazzano") | .subcomponents[] | select (.name == "verrazzano") | .images[] | select ((.image == "console") or (.image == "console-jenkins")) | .tag')
  # Split tag on '-' character and get last element
  console_sha=${console_tag##*-}

  if [[ -z $console_sha ]]; then
    echo "Failed to determine console SHA from Verrazzano BOM"
    exit 1
  fi
fi

echo "Using console commit(-ish) at $console_sha"

# Do a git clone of the console repo and checkout the commit sha (or branch or tag) provided
cd ${GO_REPO_PATH}
echo "Current dir $(pwd)"
if [ ! -e console ]; then
  echo "Cloning console repo from Github"
  git clone https://github.com/verrazzano/console.git
  cd console
else
  # for local runs typically
  CONSOLE_BRANCH=$(git branch --show-current)
  echo "Repo exists, current branch: ${CONSOLE_BRANCH}"
  cd console
  git fetch --all
  git pull
fi
git checkout ${console_sha}

# Run the basic UI tests, and if they fail make sure to exit with a fail status
make run-ui-tests
rc=$?
if [ "${rc}" != "0" ]; then
  sh "${GO_REPO_PATH}/verrazzano/ci/scripts/save_console_test_artifacts.sh"
  git checkout ${CONSOLE_BRANCH}
  cleanupOnError $?
fi

# Run the application page UI tests if specified
if [ "true" == "${RUN_APP_TESTS}" ]; then
  echo "Running Application Page UI tests"
  make run-app-page-test
  cleanupOnError $?
fi
