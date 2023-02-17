#!/usr/bin/env bash
#
# Copyright (c) 2021, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# login
/opt/keycloak/bin/kcadm.sh config credentials --server http://localhost:8080/auth --realm master --user keycloakadmin --password ##KEYCLOAK_PASSWORD##

#create realm
/opt/keycloak/bin/kcadm.sh create realms -s realm=##REALM_NAME## -s enabled=false

# create a user
/opt/keycloak/bin/kcadm.sh create users -r ##REALM_NAME## -s username=testuser -s enabled=true

# set user password
/opt/keycloak/bin/kcadm.sh set-password -r ##REALM_NAME## --username testuser --new-password ##REALM_USER_PASSWORD##

# create a keycloak client
/opt/keycloak/bin/kcadm.sh create clients -r ##REALM_NAME## -s clientId=appsclient -s enabled=true -s directAccessGrantsEnabled=true -s publicClient=true

# create a role
/opt/keycloak/bin/kcadm.sh create roles -r ##REALM_NAME## -s name=customer

# map user to role
/opt/keycloak/bin/kcadm.sh add-roles -r ##REALM_NAME## --uusername testuser --rolename customer

# enable realm
/opt/keycloak/bin/kcadm.sh update realms/##REALM_NAME## -s enabled=true