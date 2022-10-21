#!/bin/bash
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#While loop for verrazzano-platform-operator to wait for webhooks to be started before starting up
while [[ "$(curl --insecure -s -o /dev/null -w '%{http_code}' https://verrazzano-platform-operator-webhook:443/validate-install-verrazzano-io-v1alpha1-verrazzano -H 'Content-Type: application/json')" != "200" ]]; do sleep 5; done
while [[ "$(curl --insecure -s -o /dev/null -w '%{http_code}' https://verrazzano-platform-operator-webhook:443/validate-install-verrazzano-io-v1beta1-verrazzano -H 'Content-Type: application/json')" != "200" ]]; do sleep 5; done
while [[ "$(curl --insecure -s -o /dev/null -w '%{http_code}' https://verrazzano-platform-operator-webhook:443/validate-clusters-verrazzano-io-v1alpha1-verrazzanomanagedcluster -H 'Content-Type: application/json')" != "200" ]]; do sleep 5; done
while [[ "$(curl --insecure -s -o /dev/null -w '%{http_code}' -XPOST https://verrazzano-platform-operator-webhook:9443/convert -H 'Content-Type: application/json' -d '{"apiVersion":"apiextensions.k8s.io/v1", "kind":"ConversionReview", "request":{}}')" != "200" ]]; do sleep 5; done