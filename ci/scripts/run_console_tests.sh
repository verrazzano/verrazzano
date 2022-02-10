#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [ -z "$KUBECONFIG" ] || [ -z "$GO_REPO_PATH" ] || [ -z "${DOCKER_CREDS_USR}" ] ||
 [ -z "${DOCKER_CREDS_PSW}" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi



# Temporarily clone the console repo until it is moved to the Verrazzano repo
cd ${GO_REPO_PATH}
console_sha=${CONSOLE_REPO_BRANCH}
# if no branch override was supplied, use the console SHA present in the Verrazzano BOM
if [[ -z $console_sha ]]; then
  # Get tag of console from BOM
  # shellcheck disable=SC2002
  console_tag=$(cat "verrazzano/platform-operator/verrazzano-bom.json" |jq -r '.components[] | select(.name == "verrazzano") | .subcomponents[] | select (.name == "verrazzano") | .images[] | select (.image == "console") | .tag')
  # Split tag on '-' character and get last element
  console_sha=${console_tag##*-}

  if [[ -z $console_sha ]]; then
    echo "Failed to determine console SHA from Verrazzano BOM"
    exit 1
  fi
fi

echo "Using console commit at $console_sha"

# Download/Unzip console repo
zip_download_name="${console_sha}.zip"
if [ ! -d console ]; then
  wget "https://github.com/verrazzano/console/archive/$zip_download_name"
  unzip "$zip_download_name" -d console
  mv console/*/* console
fi
cd console

# Run the basic UI tests, and if they fail make sure to exit with a fail status
make run-ui-tests || exit 1

# Run the application page UI tests if specified
if [ "true" == "${RUN_APP_TESTS}" ]; then
  echo "Running Application Page UI tests"
  make run-app-page-test
fi