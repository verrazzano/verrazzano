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
GIT_HISTORY_LENGTH=$(git log $YML_FILENAME | wc -l)
if [ "$GIT_HISTORY_LENGTH" -eq 0 ] ; then
  # This is a new file - just use boilerplate copyright header
  echo "Adding YAML header to new file $YML_FILENAME"
  cat $YML_HEADER $YML_FILENAME > $TMP_YML
  mv $TMP_YML $YML_FILENAME
else
  # This is an existing file - get and use its existing copyright header by getting first 2 lines
  # of file at most recent revision
  echo "Adding back YAML header to existing file $YML_FILENAME"
  git show HEAD~1:$YML_FILENAME | head -2 > $TMP_YML
  cat $YML_FILENAME | sed 1,2d >> $TMP_YML
  mv $TMP_YML $YML_FILENAME
fi
