#!/bin/bash
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#While loop for verrazzano-platform-operator to wait for webhooks to be started before starting up
while [[ "$(curl --insecure -s -o /dev/null -w '%{http_code}' https://verrazzano-application-operator-webhook:443/validate-clusters-verrazzano-io-v1alpha1-multiclusterapplicationconfiguration -H 'Content-Type: application/json')" != "200" ]]; do sleep 5; done
while [[ "$(curl --insecure -s -o /dev/null -w '%{http_code}' https://verrazzano-application-operator-webhook:443/validate-clusters-verrazzano-io-v1alpha1-multiclustercomponent -H 'Content-Type: application/json')" != "200" ]]; do sleep 5; done
while [[ "$(curl --insecure -s -o /dev/null -w '%{http_code}' https://verrazzano-application-operator-webhook:443/validate-clusters-verrazzano-io-v1alpha1-multiclusterconfigmap -H 'Content-Type: application/json')" != "200" ]]; do sleep 5; done
while [[ "$(curl --insecure -s -o /dev/null -w '%{http_code}' https://verrazzano-application-operator-webhook:443/validate-clusters-verrazzano-io-v1alpha1-multiclustersecret -H 'Content-Type: application/json')" != "200" ]]; do sleep 5; done
while [[ "$(curl --insecure -s -o /dev/null -w '%{http_code}' https://verrazzano-application-operator-webhook:443/validate-clusters-verrazzano-io-v1alpha1-verrazzanoproject -H 'Content-Type: application/json')" != "200" ]]; do sleep 5; done
while [[ "$(curl --insecure -s -o /dev/null -w '%{http_code}' https://verrazzano-application-operator-webhook:443/appconfig-defaulter -H 'Content-Type: application/json')" != "200" ]]; do sleep 5; done
while [[ "$(curl --insecure -s -o /dev/null -w '%{http_code}' https://verrazzano-application-operator-webhook:443/validate-oam-verrazzano-io-v1alpha1-ingresstrait -H 'Content-Type: application/json')" != "200" ]]; do sleep 5; done
