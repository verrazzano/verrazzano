#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Save resource version of the Keycloak pod
RV=$(kubectl get pod -l app.kubernetes.io/name=keycloak -n keycloak -o jsonpath="{.items[0].metadata.resourceVersion}")

# Kill the MySQL pod, this will cause the Keycloak configuration to be lost
POD=$(kubectl get pod -l app=mysql -n keycloak -o jsonpath="{.items[0].metadata.name}" 2>/dev/null)
if [ $? -ne 0 ] ; then
  POD=$(kubectl get pod -l tier=mysql -n keycloak -o jsonpath="{.items[0].metadata.name}")
fi
echo "Killing pod $POD"
kubectl delete pod -n keycloak "$POD"

# Wait for MySQL to restart
POD=$(kubectl get pod -l app=mysql -n keycloak -o jsonpath="{.items[0].metadata.name}" 2>/dev/null)
if [ $? -ne 0 ] ; then
  POD=$(kubectl get pod -l tier=mysql -n keycloak -o jsonpath="{.items[0].metadata.name}")
fi
echo "Waiting for $POD to be ready"
kubectl -n keycloak wait --for=condition=ready --timeout=600s pod/"$POD"

# Wait for Keycloak configuration to be healthy.  The VPO will rebuild the Keycloak configuration.
secret=$(kubectl get secret --namespace keycloak keycloak-http -o jsonpath="{.data.password}" | base64 --decode; echo)
ingress=$(kubectl get ingress -n keycloak keycloak -o jsonpath="{.spec.rules[0].host}")

response=0
echo "Waiting for the keycloak configuration to be healthy ..."
until [ $response -eq 200 ]
do
    sleep 10
    token=$(curl -k --data "username=keycloakadmin&password=$secret&grant_type=password&client_id=admin-cli" https://"$ingress"/auth/realms/master/protocol/openid-connect/token | jq -r '.access_token')
    response=$(curl -o /dev/null -s -w "%{http_code}\n" -k -H  "Authorization: bearer $token" -X GET https://"$ingress"/auth/admin/realms/verrazzano-system/groups)
done
echo "Keycloak configuration is healthy"
