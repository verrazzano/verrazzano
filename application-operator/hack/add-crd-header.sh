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

for CRD_FILENAME in $GENERATED_CRDS_DIR/*.y*ml ; do
  GIT_HISTORY_LENGTH=$(git log $CRD_FILENAME | wc -l)
  if [ "$GIT_HISTORY_LENGTH" -eq 0 ] ; then
    echo "Adding header from $CRD_HEADER to generated NEW CRD file $CRD_FILENAME"
    TMP_CRD=${CRD_FILENAME}.tmp
    cat $CRD_HEADER $CRD_FILENAME > $TMP_CRD
    mv $TMP_CRD $CRD_FILENAME
  else
    echo "Adding back previous header to re-generated existing file $CRD_FILENAME"
    # get and use existing copyright header by getting first 2 lines of file at
    # most recent revision
    TMP_CRD=${CRD_FILENAME}.tmp
    # Some platforms get an error 141 when piping directly to head command.
    # The work around is to write the 'git show' output to a file and process that file with head.
    git show HEAD:"$CRD_FILENAME" > ${TMP_CRD}.full
    head -2 ${TMP_CRD}.full > "$TMP_CRD"
    cat "$CRD_FILENAME" >> "$TMP_CRD"
    rm  ${TMP_CRD}.full
    mv $TMP_CRD $CRD_FILENAME
  fi
done
