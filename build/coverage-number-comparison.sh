#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

#Get latest line-rate from master/release for comparison
OBJECT_URL=https://objectstorage.us-phoenix-1.oraclecloud.com/n/stevengreenberginc/b/verrazzano-builds/o/abehern/vz-7560-Enforce-UT-branch-coverage-gt-or-eq-master/unit-test-coverage-number.txt
COV_TXT=unit-test-coverage-number.txt
COV_XML=coverage.xml

EXIT_STATUS=$?

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
  BRANCH_LINE_RATE=$(grep -i '<coverage' "$COV_XML" | awk -F' ' '{print $2}' | \
    sed -E 's/line-rate=\"(.*)\"/\1/')

  RATE=$(echo "$BRANCH_LINE_RATE >= $MASTER_LINE_RATE" | bc)
  if [ "$RATE" -eq 1 ]
  then
    echo "Branch-line-rate: $BRANCH_LINE_RATE is gte to Master-line-rate: $MASTER_LINE_RATE"
    echo "Writing $BRANCH_LINE_RATE to $COV_TXT"
    echo "$BRANCH_LINE_RATE" > "$COV_TXT"
    echo 0 > "$EXIT_TXT"

  else
    echo "WARNING: Unit Test coverage(line-rate) does NOT pass"
    echo "Branch-line-rate: $BRANCH_LINE_RATE is lte to Master-line-rate: $MASTER_LINE_RATE"
    echo 1 > "$EXIT_TXT"
  fi
}
compare-coverage-numbers
