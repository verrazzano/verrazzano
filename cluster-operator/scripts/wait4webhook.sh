#!/bin/bash
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
# While loop for verrazzano-cluster-operator to wait for webhooks to be started before starting up
while [[ "$(curl --insecure -s -o /dev/null -w '%{http_code}' https://verrazzano-cluster-operator-webhook:443//validate-clusters-verrazzano-io-v1alpha1-verrazzanomanagedcluster:q
 :q
 -H 'Content-Type: application/json')" != "200" ]]; do sleep 5; done
