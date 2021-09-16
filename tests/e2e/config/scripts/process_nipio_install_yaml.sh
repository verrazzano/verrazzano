#!/bin/bash

# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

INSTALL_CONFIG_TO_EDIT=$1
DNS_WILDCARD_DOMAIN=${2:-"nip.io"}
echo "Editing install config file for DNS Wildcard domain (e.g. nip.io) ${INSTALL_CONFIG_TO_EDIT}"
yq -i eval ".spec.environmentName = \"${VZ_ENVIRONMENT_NAME}\"" ${INSTALL_CONFIG_TO_EDIT}
yq -i eval ".spec.profile = \"${INSTALL_PROFILE}\"" ${INSTALL_CONFIG_TO_EDIT}
if [ $INSTALL_PROFILE == "dev" ]; then
  yq -i eval ".spec.components.keycloak.mysql.mysqlInstallArgs.[0].name = \"persistence.enabled\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.keycloak.mysql.mysqlInstallArgs.[0].value = \"false\"" ${INSTALL_CONFIG_TO_EDIT}
fi
yq -i eval ".spec.components.dns.wildcard.domain = \"${DNS_WILDCARD_DOMAIN}\"" ${INSTALL_CONFIG_TO_EDIT}

if [ $VZ_ENVIRONMENT_NAME == "admin" ] && [ $EXTERNAL_ELASTICSEARCH == "true" ]; then
  EXTERNAL_ES_SECRET=external-es-secret
  # TODO how to get nginx ingress IP before nginx installation
  EXTERNAL_ES_URL=https://external-es.default.172.18.0.232.nip.io
  yq -i eval ".spec.components.fluentd.elasticsearchSecret = \"${EXTERNAL_ES_SECRET}\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.fluentd.elasticsearchURL = \"${EXTERNAL_ES_URL}\"" ${INSTALL_CONFIG_TO_EDIT}
fi

cat ${INSTALL_CONFIG_TO_EDIT}
