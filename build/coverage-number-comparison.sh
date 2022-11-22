#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

#Get latest line-rate from master/release for comparison
COV_TXT=unit-test-coverage-number.txt
LOCAL_BRANCH_LINE_RATE=$(grep -i '<coverage' coverage.xml | awk -F' ' '{print $2}' | sed -E 's/line-rate=\"(.*)\"/\1/')
LOCAL_BRANCH_VERSION=$(grep -i 'version=' .verrazzano-development-version | awk -F '=' '{print $2}' | head -c3)
LOCAL_BRANCH_NAME=$(git branch  --no-color  | grep -E '^\*' | sed 's/\*[^a-z]*//g')


#Returns 1 if Local-line-rate is gte Remote-line-rate, otherwise returns 0
compareCoverageNumbers(){
  REMOTE_LINE_RATE=$(cat "$COV_TXT")
  RATE="$LOCAL_BRANCH_LINE_RATE >= $REMOTE_LINE_RATE"
  RESULT=$(echo "$RATE" | bc)
    if [[ "$RESULT" -eq 1 ]]
        then
          echo "PASS."
          echo "Branch-line-rate: $LOCAL_BRANCH_LINE_RATE is gte to Remote-line-rate"
#          echo "$LOCAL_BRANCH_LINE_RATE" > "$COV_TXT"
        else
          echo "WARNING: Unit Test coverage(line-rate) does NOT pass"
          echo "Branch-line-rate: $LOCAL_BRANCH_LINE_RATE is lte to Remote-line-rate"
          echo "Job Fail."
          exit 1
    fi
}

#Determine if on a release or master branch and download unit-test-coverage-number
if [[ ! "$LOCAL_BRANCH_NAME" =~ ^release-[0-9]+\.[0-9]+$|^master$ ]]
then
    echo "Trying to download unit-test-coverage-number.txt from release-$LOCAL_BRANCH_VERSION..."
    oci --region us-phoenix-1 os object get --namespace "$OCI_OS_NAMESPACE" -bn "$OCI_OS_BUCKET" --name release-"$LOCAL_BRANCH_VERSION"/unit-test-coverage-number.txt --file unit-test-coverage-number.txt

    if [[ $? -gt 0  ]];
    then
      echo "Trying to download unit-test-coverage-number.txt from master..."
      oci --region us-phoenix-1 os object get --namespace "$OCI_OS_NAMESPACE" -bn "$OCI_OS_BUCKET" --name master/unit-test-coverage-number.txt -file unit-test-coverage-number.txt
    fi

    #Runs when we are on a feature branch and determines if line coverage passes
    compareCoverageNumbers

else
  echo "Is a release-* or master branch..."
  echo "Putting unit-test-coverage-number.txt into object at $CLEAN_BRANCH_NAME/unit-test-coverage-number.txt"
  oci --region us-phoenix-1 os object put --force --namespace "$OCI_OS_NAMESPACE" -bn "$OCI_OS_BUCKET" --name "$CLEAN_BRANCH_NAME"/unit-test-coverage-number.txt --file unit-test-coverage-number.txt
fi
