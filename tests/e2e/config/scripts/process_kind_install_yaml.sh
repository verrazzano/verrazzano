#!/bin/bash
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Requires the env var INSTALL_CONFIG_FILE_KIND be defined at a minimum for KIND installs
#
# Optional:
# VZ_ENVIRONMENT_NAME - environmentName default
# DNS_WILDCARD_DOMAIN - wildcard DNS domain to use
# EXTERNAL_ELASTICSEARCH - if "true" && VZ_ENVIRONMENT_NAME=="admin", sets Fluentd configuration to point to EXTERNAL_ES_SECRET and EXTERNAL_ES_URL
# SYSTEM_LOG_ID - configures Verrazzano for OCI logging using the specified OCI logging ID
#
INSTALL_CONFIG_TO_EDIT=$1
DNS_WILDCARD_DOMAIN=${2:-""}
INSTALL_PROFILE=${INSTALL_PROFILE:-"dev"}
API_VERSION="v1beta1"

if [ -z "${INSTALL_CONFIG_TO_EDIT}" ]; then
  echo "Please pass in a valid Verrazzano configuration file"
fi

if [ -z "${CRD_API_VERSION}" ]; then
  echo "CRD_API_VERSION is not defined so using default API version to v1beta1"
elif [ "$CRD_API_VERSION" == "v1alpha1" ]; then
  API_VERSION="v1alpha1"
fi

echo "Editing install config file for kind ${INSTALL_CONFIG_TO_EDIT}"
yq -i eval ".spec.profile = \"${INSTALL_PROFILE}\"" ${INSTALL_CONFIG_TO_EDIT}
if [ -n "${VZ_ENVIRONMENT_NAME}" ]; then
  yq -i eval ".spec.environmentName = \"${VZ_ENVIRONMENT_NAME}\"" ${INSTALL_CONFIG_TO_EDIT}
fi
if [ -n "${DNS_WILDCARD_DOMAIN}" ]; then
  yq -i eval ".spec.components.dns.wildcard.domain = \"${DNS_WILDCARD_DOMAIN}\"" ${INSTALL_CONFIG_TO_EDIT}
fi

if [ "$VZ_ENVIRONMENT_NAME" == "admin" ] && [ "$EXTERNAL_ELASTICSEARCH" == "true" ] && [ "$API_VERSION" == "v1alpha1" ]; then
  EXTERNAL_ES_SECRET=external-es-secret
  EXTERNAL_ES_URL=https://$(KUBECONFIG=${ADMIN_KUBECONFIG} kubectl get svc opensearch-cluster-master -o=jsonpath='{.status.loadBalancer.ingress[0].ip}'):9200
  yq -i eval ".spec.components.fluentd.elasticsearchSecret = \"${EXTERNAL_ES_SECRET}\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.fluentd.elasticsearchURL = \"${EXTERNAL_ES_URL}\"" ${INSTALL_CONFIG_TO_EDIT}
elif [ "$VZ_ENVIRONMENT_NAME" == "admin" ] && [ "$EXTERNAL_ELASTICSEARCH" == "true" ] && [ "$API_VERSION" == "v1beta1" ]; then
  EXTERNAL_ES_SECRET=external-es-secret
  EXTERNAL_ES_URL=https://$(KUBECONFIG=${ADMIN_KUBECONFIG} kubectl get svc opensearch-cluster-master -o=jsonpath='{.status.loadBalancer.ingress[0].ip}'):9200
  yq -i eval ".spec.components.fluentd.opensearchSecret = \"${EXTERNAL_ES_SECRET}\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.fluentd.opensearchURL = \"${EXTERNAL_ES_URL}\"" ${INSTALL_CONFIG_TO_EDIT}
fi

if [ -n "${SYSTEM_LOG_ID}" ] && [ $API_VERSION == "v1alpha1" ]; then
  yq -i eval ".spec.components.fluentd.oci.systemLogId = \"${SYSTEM_LOG_ID}\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.fluentd.oci.defaultAppLogId = \"${APP_LOG_ID}\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.fluentd.oci.apiSecret = \"oci-fluentd\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.elasticsearch.enabled = false" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.kibana.enabled = false" ${INSTALL_CONFIG_TO_EDIT}
elif [ -n "${SYSTEM_LOG_ID}" ] && [ $API_VERSION == "v1beta1" ]; then
  yq -i eval ".spec.components.fluentd.oci.systemLogId = \"${SYSTEM_LOG_ID}\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.fluentd.oci.defaultAppLogId = \"${APP_LOG_ID}\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.fluentd.oci.apiSecret = \"oci-fluentd\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.opensearch.enabled = false" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.opensearchDashboards.enabled = false" ${INSTALL_CONFIG_TO_EDIT}
fi

echo """
Verrazzano configuration:

"""
cat ${INSTALL_CONFIG_TO_EDIT}
