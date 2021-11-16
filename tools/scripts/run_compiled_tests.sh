#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# This script can be used to execute a compiled test binary against a verrazzano cluster and (optionally) push the test logs
# and report to objectstorage.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
function usage {
    echo
    echo "usage: $0 -t test_binary_url_or_file -k kubeconfig_text -c kubeconfig_url_or_file -r regex_filter_for_tests -o test_logs_os_bucket -l test_log_archive_name"
    echo "  -t test_binary_url_or_file The url or path of test binary file"
    echo "  -k kubeconfig_text         kubeconfig text"
    echo "  -c kubeconfig_url_or_file  The url or path of kubeconfig"
    echo "  -r regex_filter_for_tests  The regex filter for tests"
    echo "  -o test_logs_os_bucket     The OS Bucket name for test logs, please make sure OCI CLI env variables are set."
    echo "  -l test_log_archive_name   The name of test log archive file"
    echo "  -h help                    show this help"
    echo
    exit 1
}

is_url() {
    regex='^(https?|ftp|file)://[-A-Za-z0-9\+&@#/%?=~_|!:,.;]*[-A-Za-z0-9\+&@#/%=~_|]\.[-A-Za-z0-9\+&@#/%?=~_|!:,.;]*[-A-Za-z0-9\+&@#/%=~_|]$'
    [[ $1 =~ $regex ]]
}

file_exists_at_url() {
    wget --spider $1 2>/dev/null
}

file_exists() {
    [[ -f "$1" && -s "$1" ]]
}

TEST_BINARY=""
KUBECONFIG_TEXT=""
KUBECONFIG_LOCATION=""
TEST_REGEX=""
TEST_LOG_BUCKET=""
TEST_LOG_ARCHIVE=""
TEST_REGEX_DEFAULT=".*"
TS=`date "+%Y%m%d-%H%M%S%ss"`
TEST_LOG_ARCHIVE_DEFAULT="test-output-${TS}"


while getopts t:k:c:r:o:l:h flag
do
    case "${flag}" in
        t) TEST_BINARY=${OPTARG};;
        k) KUBECONFIG_TEXT=${OPTARG};;
        c) KUBECONFIG_LOCATION=${OPTARG};;
        r) TEST_REGEX=${OPTARG};;
        o) TEST_LOG_BUCKET=${OPTARG};;
        l) TEST_LOG_ARCHIVE=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

if [ -z "${TEST_BINARY}" ] ; then
    echo "TEST_BINARY must be set!"
    exit 1
fi

if is_url "${TEST_BINARY}"
then
    if file_exists_at_url "${TEST_BINARY}"
    then
      FILE_NAME=`basename "${TEST_BINARY}"`
      rm -rf /tmp/$FILE_NAME
      wget "${TEST_BINARY}" -O "/tmp/$FILE_NAME"
      TEST_BINARY="/tmp/$FILE_NAME"
    else
      echo "No file exists at url specified by ${TEST_BINARY}!"
      exit 1
    fi
fi

if file_exists "${TEST_BINARY}"
then
  echo "Using TEST Binary fie at ${TEST_BINARY}"
else
  echo "No file exists at ${TEST_BINARY} or is empty!"
  exit 1
fi

if is_url "${KUBECONFIG_LOCATION}"
then
    if file_exists_at_url "${KUBECONFIG_LOCATION}"
    then
      FILE_NAME=`basename "${KUBECONFIG_LOCATION}"`
      rm -rf /tmp/$FILE_NAME
      wget "${KUBECONFIG_LOCATION}" -O "/tmp/$FILE_NAME"
      KUBECONFIG_LOCATION="/tmp/$FILE_NAME"
    else
      echo "No file exists at url specified by ${KUBECONFIG_LOCATION}!"
      exit 1
    fi
fi

if file_exists "${KUBECONFIG_LOCATION}"
then
  echo "Using Kubeconfig at ${KUBECONFIG_LOCATION}"
else
  if [ -z "${KUBECONFIG_TEXT}" ]
  then
    if [ ! -z "${KUBECONFIG}" ]
    then
      echo "Using KUBECONFIG from env variable with value ${KUBECONFIG}"
      KUBECONFIG_LOCATION="${KUBECONFIG}"
    else
      echo "KUBECONFIG_TEXT must be set or KUBECONFIG_LOCATION should point to a valid kubeconfig or there should be a valid KUBECONFIG env variable!"
      exit 1
    fi
  else
    KUBECONFIG_LOCATION="/tmp/kubeconfig"
    rm -rf "${KUBECONFIG_LOCATION}"
    touch "${KUBECONFIG_LOCATION}"
    echo "${KUBECONFIG_TEXT}" > "${KUBECONFIG_LOCATION}"
  fi  
fi

if [ -z "${TEST_REGEX}" ]; then
  TEST_REGEX="${TEST_REGEX_DEFAULT}"
fi

if [ -z "${TEST_LOG_ARCHIVE}" ]; then
  TEST_LOG_ARCHIVE="${TEST_LOG_ARCHIVE_DEFAULT}"
fi

rm -rf /tmp/${TEST_LOG_ARCHIVE}
mkdir -p /tmp/${TEST_LOG_ARCHIVE}
touch /tmp/${TEST_LOG_ARCHIVE}/test.log
chmod a+x "${KUBECONFIG_LOCATION}"
chmod a+x "${TEST_BINARY}"
export KUBECONFIG="${KUBECONFIG_LOCATION}"
# Remove a warning that suggests using Ginkgo 2.0
export ACK_GINKGO_DEPRECATIONS=1.16.5
ginkgo -v -keepGoing --reportFile="/tmp/${TEST_LOG_ARCHIVE}/test.report" -outputdir="/tmp/${TEST_LOG_ARCHIVE}" --trace --focus="${TEST_REGEX}" "${TEST_BINARY}" | tee /tmp/${TEST_LOG_ARCHIVE}/test.log

if [ ! -z "${TEST_LOG_BUCKET}" ]; then
  tar -czvf /tmp/${TEST_LOG_ARCHIVE}.tgz /tmp/${TEST_LOG_ARCHIVE}/*
  # When this script is run in a k8s job, it expects the OCI creds to be present in /etc/ocisecret/oci.yaml. 
  if file_exists "/etc/ocisecret/oci.yaml"
  then
    export OCI_CLI_PROFILE="DEFAULT"
    export OCI_CLI_USER=`yq e .auth.user /etc/ocisecret/oci.yaml`
    export OCI_CLI_FINGERPRINT=`yq e .auth.fingerprint /etc/ocisecret/oci.yaml`
    yq e .auth.key /etc/ocisecret/oci.yaml > /tmp/ocikey.pem
    export OCI_CLI_KEY_FILE=/tmp/ocikey.pem
    export OCI_CLI_TENANCY=`yq e .auth.tenancy /etc/ocisecret/oci.yaml`
    export OCI_CLI_REGION=`yq e .auth.region /etc/ocisecret/oci.yaml`
    export OCI_CLI_SUPPRESS_FILE_PERMISSIONS_WARNING="True"
  fi
  oci os object put --bucket-name "${TEST_LOG_BUCKET}" --file /tmp/${TEST_LOG_ARCHIVE}.tgz
fi

status=`sed -n -e '$p' /tmp/${TEST_LOG_ARCHIVE}/test.log`
if [ "$status" == "Test Suite Failed" ]
then
  echo "Tests Failed."
  exit 1
else
  echo "Tests Passed"
  exit 0
fi