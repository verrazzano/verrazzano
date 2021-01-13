#!/bin/bash
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# Add YAML boilerplate to generated CRDs - kubebuilder currently does not seem to have a way to
# add boilerplate headers to these - only to generated Go files

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname $0)/..
YML_HEADER=$(dirname $0)/boilerplate.yaml.txt
YML_FILENAME=$SCRIPT_ROOT/$1

TMP_YML=${YML_FILENAME}.tmp
cat $YML_HEADER $YML_FILENAME > $TMP_YML
mv $TMP_YML $YML_FILENAME
