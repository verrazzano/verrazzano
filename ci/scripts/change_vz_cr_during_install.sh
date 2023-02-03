#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# TODO: write description for environment variables and script arguments here

# wait for vz resource to exist
while ! kubectl get vz
do
    echo "Waiting for verrazzano resource to be created..."
    sleep 30s
done

# wait a specified amount of time. FIXME: Maybe make this an argument to this script
wait_min=2
echo "Waiting for ${wait_min} minutes..."

# TODO: change the Wildcard DNS domain. Maybe make this flexible depending on script arguments? FIXME: use variable for nip.io
kubectl patch vz my-verrazzano -p '{"spec":{"components":{"dns":{"wildcard":{"domain":"nip.io"}}}}}' --type=merge

# TODO: maybe exit with error depnding on conditions?
exit 0