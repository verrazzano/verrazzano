#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

TEST_CONFIG_FILE=$1
DNS_ZONE=$2
sed -i "s/XX_DNS_ZONE_XX/${DNS_ZONE}/" ${TEST_CONFIG_FILE}
