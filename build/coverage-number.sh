#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

#Extracts the line-rate number from coverage.xml
grep -i '<coverage' coverage.xml | awk -F' ' '{print $2}' | \
sed -E 's/line-rate=\"(.*)\"/\1/' > unit-test-coverage-number.txt