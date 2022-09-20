#!/bin/bash

# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

INSTALL_CONFIG_TO_EDIT=$1
DNS_WILDCARD_DOMAIN=${2:-"nip.io"}
echo "Editing install config file for DNS Wildcard domain (e.g. nip.io) ${INSTALL_CONFIG_TO_EDIT}"
yq -i eval ".spec.environmentName = \"${VZ_ENVIRONMENT_NAME}\"" ${INSTALL_CONFIG_TO_EDIT}
yq -i eval ".spec.profile = \"${INSTALL_PROFILE}\"" ${INSTALL_CONFIG_TO_EDIT}
if [ $INSTALL_PROFILE == "dev" ] && [ $CRD_API_VERSION == "v1alpha1" ]; then
  yq -i eval ".spec.components.keycloak.mysql.mysqlInstallArgs.[0].name = \"persistence.enabled\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.keycloak.mysql.mysqlInstallArgs.[0].value = \"false\"" ${INSTALL_CONFIG_TO_EDIT}
elif [ $INSTALL_PROFILE == "dev" ] && [ $CRD_API_VERSION == "v1beta1" ]; then
  yq -i eval ".spec.components.keycloak.overrides.[0].values.persistence.enabled = false" ${INSTALL_CONFIG_TO_EDIT}
fi
yq -i eval ".spec.components.dns.wildcard.domain = \"${DNS_WILDCARD_DOMAIN}\"" ${INSTALL_CONFIG_TO_EDIT}

if [ -n "${SYSTEM_LOG_ID}" ] && [ $CRD_API_VERSION == "v1alpha1" ]; then
  yq -i eval ".spec.components.fluentd.oci.systemLogId = \"${SYSTEM_LOG_ID}\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.fluentd.oci.defaultAppLogId = \"${APP_LOG_ID}\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.elasticsearch.enabled = false" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.kibana.enabled = false" ${INSTALL_CONFIG_TO_EDIT}
elif [ -n "${SYSTEM_LOG_ID}" ] && [ $CRD_API_VERSION == "v1beta1" ]; then
  yq -i eval ".spec.components.fluentd.oci.systemLogId = \"${SYSTEM_LOG_ID}\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.fluentd.oci.defaultAppLogId = \"${APP_LOG_ID}\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.opensearch.enabled = false" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.opensearchDashboards.enabled = false" ${INSTALL_CONFIG_TO_EDIT}
fi
cat ${INSTALL_CONFIG_TO_EDIT}
