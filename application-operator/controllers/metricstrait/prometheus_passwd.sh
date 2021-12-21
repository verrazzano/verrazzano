#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

POD_NAME=$(kubectl get pods -n verrazzano-system | grep prometheus | awk '{print $2}')

kubectl cp /tmp/password_file.txt verrazzano-system/$POD_NAME:etc/prometheus/password_file.txt
