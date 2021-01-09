#!/bin/bash
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# Add YAML boilerplate to generated CRDs - kubebuilder currently does not seem to have a way to
# add boilerplate headers to these - only to generated Go files

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname $0)/..
CRD_HEADER=$(dirname $0)/boilerplate.yaml.txt
GENERATED_CRDS_DIR=$SCRIPT_ROOT/config/crd/bases

for CRD_FILENAME in $(ls $GENERATED_CRDS_DIR/*.y*ml) ; do
  echo "Adding header from $CRD_HEADER to generated CRD file $CRD_FILENAME"
  TMP_CRD=${CRD_FILENAME}.tmp
  cat $CRD_HEADER $CRD_FILENAME > $TMP_CRD
  mv $TMP_CRD $CRD_FILENAME
done
