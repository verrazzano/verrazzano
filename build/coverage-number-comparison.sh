#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

#Get latest line-rate from master/release for comparison
LOCAL_BRANCH_NAME=$(git branch  --no-color  | grep -E '^\*' | sed 's/\*[^a-z]*//g')
OBJECT_URL=https://objectstorage.us-phoenix-1.oraclecloud.com/n/stevengreenberginc/b/verrazzano-builds/o/"$LOCAL_BRANCH_NAME"/unit-test-coverage-number.txt
COV_TXT=unit-test-coverage-number.txt
COV_XML=coverage.xml


if [ ! -f "$COV_TXT" ]
then
  echo "File: "$COV_TXT" does NOT exist."
  echo "DOWNLOADING..."
  wget "$OBJECT_URL"
else
  echo "File: "$COV_TXT" already exists."
fi

compare-coverage-numbers(){
  MASTER_LINE_RATE=$(cat "$COV_TXT")
  BRANCH_LINE_RATE=$(grep -i '<coverage' "$COV_XML" | awk -F' ' '{print $2}' | sed -E 's/line-rate=\"(.*)\"/\1/')

  RATE=$(echo "$BRANCH_LINE_RATE >= $MASTER_LINE_RATE" | bc)
  if [ "$RATE" -eq 1 ]
  then
    echo "Branch-line-rate: $BRANCH_LINE_RATE is gte to Master-line-rate: $MASTER_LINE_RATE"
    echo "Writing $BRANCH_LINE_RATE to $COV_TXT"
    echo "$BRANCH_LINE_RATE" > "$COV_TXT"
    echo "Putting " "$COV_TXT" " into OCI Object Storage..."
    oci --region us-phoenix-1 os object put --force --namespace "$OCI_OS_NAMESPACE" -bn "$OCI_OS_BUCKET" --name "$LOCAL_BRANCH_NAME"/unit-test-coverage-number.txt --file unit-test-coverage-number.txt
    exit 0

  else
    echo "WARNING: Unit Test coverage(line-rate) does NOT pass"
    echo "Branch-line-rate: $BRANCH_LINE_RATE is lte to Master-line-rate: $MASTER_LINE_RATE"
    exit 1
  fi
}
compare-coverage-numbers
