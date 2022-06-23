#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Provide private-ip-ocid to fetch the associated ip-address 
oci network private-ip get --private-ip-id $1 | jq -r '.data."ip-address"'