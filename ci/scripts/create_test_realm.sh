#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

userpass="${REALM_USER_PASSWORD:-testuserpass}"

realmName="${REALM_NAME:-test-realm}"

# login
/opt/jboss/keycloak/bin/kcadm.sh config credentials --server http://localhost:8080/auth --realm master --user keycloakadmin --password ${KEYCLOAK_PASSWORD}

#create realm
/opt/jboss/keycloak/bin/kcadm.sh create realms -s realm=${realmName} -s enabled=false

# create a user
/opt/jboss/keycloak/bin/kcadm.sh create users -r ${realmName} -s username=testuser -s enabled=true

# set user password
/opt/jboss/keycloak/bin/kcadm.sh set-password -r ${realmName} --username testuser --new-password ${userpass}

# create a keycloak client
/opt/jboss/keycloak/bin/kcadm.sh create clients -r ${realmName} -s clientId=appsclient -s enabled=true -s directAccessGrantsEnabled=true -s publicClient=true

# create a role
/opt/jboss/keycloak/bin/kcadm.sh create roles -r ${realmName} -s name=customer

# map user to role
/opt/jboss/keycloak/bin/kcadm.sh add-roles -r ${realmName} --uusername testuser --rolename customer

# enable realm
/opt/jboss/keycloak/bin/kcadm.sh update realms/${realmName} -s enabled=true