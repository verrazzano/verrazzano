#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

INSTALL_CONFIG_TO_EDIT=$1
DNS_WILDCARD_DOMAIN=${2:-"nip.io"}
echo "Editing install config file for kind ${INSTALL_CONFIG_TO_EDIT}"
yq -i eval ".spec.environmentName = \"${VZ_ENVIRONMENT_NAME}\"" ${INSTALL_CONFIG_TO_EDIT}
yq -i eval ".spec.profile = \"${INSTALL_PROFILE}\"" ${INSTALL_CONFIG_TO_EDIT}
yq -i eval ".spec.components.dns.wildcard.domain = \"${DNS_WILDCARD_DOMAIN}\"" ${INSTALL_CONFIG_TO_EDIT}

if [ $VZ_ENVIRONMENT_NAME == "admin" ] && [ $EXTERNAL_ELASTICSEARCH == "true" ]; then
  EXTERNAL_ES_SECRET=external-es-secret
  EXTERNAL_ES_URL=https://$(KUBECONFIG=${ADMIN_KUBECONFIG} kubectl get svc external-es-service -o=jsonpath='{.status.loadBalancer.ingress[0].ip}'):9200
  yq -i eval ".spec.components.fluentd.elasticsearchSecret = \"${EXTERNAL_ES_SECRET}\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.fluentd.elasticsearchURL = \"${EXTERNAL_ES_URL}\"" ${INSTALL_CONFIG_TO_EDIT}
fi

cat ${INSTALL_CONFIG_TO_EDIT}
