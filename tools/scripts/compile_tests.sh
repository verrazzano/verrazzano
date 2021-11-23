#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# This script can be be used to compile a test binary from a given path and (optionally) push the binary to objectstorage.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
function usage {
    echo
    echo "usage: $0 -p test_package_path -a compile_args -n test_binary_file_name -o test_binary_os_bucket"
    echo "  -p test_package_path      Absolute path to test package"
    echo "  -a compile_args           compile args to be given to ginkgo build command"
    echo "  -o test_binary_os_bucket  The OS Bucket name for test binary, please make sure OCI CLI env variables are set."
    echo "  -h help                   show this help"
    echo
    exit 1
}

TEST_PACKAGE_PATH=""
COMPILE_ARGS=""
TEST_BINARY_OS_BUCKET=""
TEST_PACKAGE_PATH_DEFAULT="."
TS=`date "+%Y%m%d-%H%M%S%s"`


while getopts p:a:o:h flag
do
    case "${flag}" in
        p) TEST_PACKAGE_PATH=${OPTARG};;
        a) COMPILE_ARGS=${OPTARG};;
        o) TEST_BINARY_OS_BUCKET=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

if [ -z "${TEST_PACKAGE_PATH}" ]; then
  TEST_PACKAGE_PATH="${TEST_PACKAGE_PATH_DEFAULT}"
fi

ginkgo build ${COMPILE_ARGS} ${TEST_PACKAGE_PATH}/... \
    | grep "compiled *" | cut -d' ' -f6 \
        | xargs -L 1 find ${TEST_PACKAGE_PATH} -name \
            | xargs -t -I {} echo {} \
                | xargs -t -I {} -L 1 sh -c "echo [ ! -z ${TEST_BINARY_OS_BUCKET} ] && echo 'oci os object put --bucket-name ${TEST_BINARY_OS_BUCKET} --file {}'" \
                    | xargs -I {} -L 1 sh -c "[ ! -z ${TEST_BINARY_OS_BUCKET} ] && {}"