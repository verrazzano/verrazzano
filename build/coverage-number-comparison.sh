#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

#Get latest line-rate from master/release for comparison
UNIT_TEST_TXT_FILE=unit-test-coverage-number.txt
LOCAL_BRANCH_LINE_RATE=$(grep -i '<coverage' coverage.xml | awk -F' ' '{print $2}' | sed -E 's/line-rate=\"(.*)\"/\1/')
LOCAL_BRANCH_VERSION=$(grep -i 'version=' .verrazzano-development-version | awk -F '=' '{print $2}' | head -c3)

#Returns 1 if Local-line-rate is gte Remote-line-rate, otherwise returns 0
compareCoverageNumbers() {
  REMOTE_LINE_RATE=$(cat "$UNIT_TEST_TXT_FILE")
  RATE="$LOCAL_BRANCH_LINE_RATE >= $REMOTE_LINE_RATE"
  RESULT=$(echo "$RATE" | bc)
  if [[ "$RESULT" -eq 1 ]]; then
    echo "PASS."
    echo "Branch-line-rate: $LOCAL_BRANCH_LINE_RATE is gte to Remote-line-rate: $REMOTE_LINE_RATE"
    uploadToObjectStorage
  else
    echo "WARNING: Unit Test coverage(line-rate) does NOT pass"
    echo "Branch-line-rate: $LOCAL_BRANCH_LINE_RATE is lte to Remote-line-rate: $REMOTE_LINE_RATE"
    if [[ "$FAIL_BUILD_COVERAGE" ]]; then
      echo "Job Failed."
      exit 1
    fi
  fi
}

#Only upload coverage number if it passes
uploadToObjectStorage() {
  if [[ "$UPLOAD_UT_COVERAGE" ]]; then
    echo "Writing new coverage number to $UNIT_TEST_TXT_FILE ..."
    echo "$LOCAL_BRANCH_LINE_RATE" > "$UNIT_TEST_TXT_FILE"
    echo "Putting in object storage at $CLEAN_BRANCH_NAME/$UNIT_TEST_TXT_FILE"
    oci --region us-phoenix-1 os object put --force --namespace "$OCI_OS_NAMESPACE" -bn "$OCI_OS_BUCKET" --name "$CLEAN_BRANCH_NAME"/"$UNIT_TEST_TXT_FILE" --file "$UNIT_TEST_TXT_FILE"
  fi
}

#Determine if on a release or master branch and download unit-test-coverage-number
if [[ ! "$CLEAN_BRANCH_NAME" =~ ^release-[0-9]+\.[0-9]+$|^master$ ]]; then
  FAIL_BUILD_COVERAGE=true
  echo "Trying to download $UNIT_TEST_TXT_FILE from release-$LOCAL_BRANCH_VERSION..."
  oci --region us-phoenix-1 os object get --namespace "$OCI_OS_NAMESPACE" -bn "$OCI_OS_BUCKET" --name release-"$LOCAL_BRANCH_VERSION"/"$UNIT_TEST_TXT_FILE" --file "$UNIT_TEST_TXT_FILE"

  if [[ $? -gt 0 ]]; then
    echo "Trying to download $UNIT_TEST_TXT_FILE from master..."
    oci --region us-phoenix-1 os object get --namespace "$OCI_OS_NAMESPACE" -bn "$OCI_OS_BUCKET" --name master/"$UNIT_TEST_TXT_FILE" --file "$UNIT_TEST_TXT_FILE"
  fi

  #Runs when we are on a feature branch and determines if line coverage passes
  compareCoverageNumbers

else
  echo "Is a release-* or master branch..."
  UPLOAD_UT_COVERAGE=true
  oci --region us-phoenix-1 os object get --namespace "$OCI_OS_NAMESPACE" -bn "$OCI_OS_BUCKET" --name "$CLEAN_BRANCH_NAME"/"$UNIT_TEST_TXT_FILE" --file "$UNIT_TEST_TXT_FILE"
  compareCoverageNumbers
fi
