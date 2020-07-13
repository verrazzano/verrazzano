#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

set -e

DNS_TYPE="xip.io"

while getopts n:d:s:h flag
do
    case "${flag}" in
        d) DNS_TYPE=${OPTARG};;
    esac
done

if [ ${DNS_TYPE} == "oci" ]; then
  "$SCRIPT_DIR"/2b-install-system-components-ocidns.sh "$@"
else
  "$SCRIPT_DIR"/2a-install-system-components-magicdns.sh "$@"
fi
