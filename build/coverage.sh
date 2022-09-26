#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Code coverage generation
# Excludes test utility packages and the test directory
TEST_PKGS=$(go list ./... | \
  grep -Ev github.com/verrazzano/verrazzano/application-operator/mocks | \
  grep -Ev github.com/verrazzano/verrazzano/application-operator/clients | \
  grep -Ev github.com/verrazzano/verrazzano/platform-operator/mocks | \
  grep -Ev github.com/verrazzano/verrazzano/platform-operator/clients | \
  grep -Ev github.com/verrazzano/verrazzano/tools/eventually-checker | \
  grep -Ev github.com/verrazzano/verrazzano/tools/fix-copyright | \
  grep -Ev github.com/verrazzano/verrazzano/tests | \
  grep -Ev github.com/verrazzano/verrazzano/pkg/test/framework)
#go test -coverpkg=$(echo ${TEST_PKGS}|tr ' ' ',') -coverprofile ./coverage.raw.cov ${TEST_PKGS}
go test -coverprofile ./coverage.raw.cov $(go list ./... | grep -v /tests/e2e)

TEST_STATUS=$?

# Remove specific files from coverage report
cat ./coverage.raw.cov |\
  grep -v "zz_generated.deepcopy" |\
  grep -v "mocks" |\
  grep -v "e2e" |\
  grep -v "generated_assets" > coverage.cov

# Display the global code coverage.  This generates the total number the badge uses
go tool cover -func=coverage.cov

# If needed, generate HTML report
if [ "$1" == "html" ]; then
    go tool cover -html=coverage.cov -o coverage.html
    GOCOV=$(command -v gocov)
    if [ $? -eq 0 ] ; then
        GOCOV_XML=$(command -v gocov-xml)
        if [  $? -eq 0 ] ; then
            $GOCOV convert coverage.cov | $GOCOV_XML > coverage.xml
        fi
    fi
fi

exit $TEST_STATUS
