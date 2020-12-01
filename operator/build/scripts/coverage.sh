#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Code coverage generation
# Excludes test utility packages and the test directory
go test -coverprofile ./coverage.cov $(go list ./... | grep -Ev github.com/verrazzano/verrazzano/operator/test)

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

