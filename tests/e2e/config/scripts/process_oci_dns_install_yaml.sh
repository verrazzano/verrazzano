#!/bin/bash

# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

INSTALL_CONFIG_TO_EDIT=$1
CERT_MGR=${2:-"acme"}
ACME_ENV=${3:-"staging"}
DNS_SCOPE=${4:-"GLOBAL"}
NGINX_LB_SCOPE=${4:-"GLOBAL"}
ISTIO_LB_SCOPE=${4:-"GLOBAL"}
if [ "${DNS_SCOPE}" != "PRIVATE" ] && [ "${DNS_SCOPE}" != "GLOBAL" ]; then
  # setting scope to global even for invalid values. This should not happen in jenkins flow
  DNS_SCOPE="GLOBAL"
fi
if [ "${NGINX_LB_SCOPE}" != "PRIVATE" ] && [ "${NGINX_LB_SCOPE}" != "GLOBAL" ]; then
  # setting scope to global even for invalid values. This should not happen in jenkins flow
  NGINX_LB_SCOPE="GLOBAL"
fi
if [ "${ISTIO_LB_SCOPE}" != "PRIVATE" ] && [ "${ISTIO_LB_SCOPE}" != "GLOBAL" ]; then
  # setting scope to global even for invalid values. This should not happen in jenkins flow
  ISTIO_LB_SCOPE="GLOBAL"
fi
echo "Editing install config file for OCI DNS ${INSTALL_CONFIG_TO_EDIT}"
yq -i eval ".spec.environmentName = \"${VZ_ENVIRONMENT_NAME}\"" ${INSTALL_CONFIG_TO_EDIT}
yq -i eval ".spec.profile = \"${INSTALL_PROFILE}\"" ${INSTALL_CONFIG_TO_EDIT}
yq -i eval ".spec.components.dns.oci.dnsZoneCompartmentOCID = \"${OCI_DNS_COMPARTMENT_OCID}\"" ${INSTALL_CONFIG_TO_EDIT}
yq -i eval ".spec.components.dns.oci.dnsZoneOCID = \"${OCI_DNS_ZONE_OCID}\"" ${INSTALL_CONFIG_TO_EDIT}
yq -i eval ".spec.components.dns.oci.dnsZoneName = \"${OCI_DNS_ZONE_NAME}\"" ${INSTALL_CONFIG_TO_EDIT}
yq -i eval ".spec.components.dns.oci.dnsScope = \"${DNS_SCOPE}\"" ${INSTALL_CONFIG_TO_EDIT}
if [ "${CERT_MGR}" == "acme" ]; then
  yq -i eval ".spec.components.certManager.certificate.acme.provider = \"letsEncrypt\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.certManager.certificate.acme.emailAddress = \"emailAddress@domain.com\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.certManager.certificate.acme.environment = \"${ACME_ENV}\"" ${INSTALL_CONFIG_TO_EDIT}
fi
if [ $INSTALL_PROFILE == "dev" ] && [ $CRD_API_VERSION == "v1alpha1" ]; then
  yq -i eval ".spec.components.keycloak.mysql.mysqlInstallArgs.[0].name = \"persistence.enabled\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.keycloak.mysql.mysqlInstallArgs.[0].value = \"false\"" ${INSTALL_CONFIG_TO_EDIT}
elif [ $INSTALL_PROFILE == "dev" ] && [ $CRD_API_VERSION == "v1beta1" ]; then
  yq -i eval ".spec.components.keycloak.overrides.[0].values.persistence.enabled = false" ${INSTALL_CONFIG_TO_EDIT}
fi

if [ "${NGINX_LB_SCOPE}" == "PRIVATE" ] && [ $CRD_API_VERSION == "v1beta1" ]; then
  yq -i eval ".spec.components.ingressNGINX.overrides[0].values.controller.service.annotations.\"service.beta.kubernetes.io/oci-load-balancer-internal\" = \"true\"" ${INSTALL_CONFIG_TO_EDIT}
fi
if [ "${ISTIO_LB_SCOPE}" == "PRIVATE" ] && [ $CRD_API_VERSION == "v1beta1" ]; then
  yq -i eval ".spec.components.istio.overrides[0].values.apiVersion = \"install.istio.io/v1alpha1\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.istio.overrides[0].values.kind = \"IstioOperator\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.istio.overrides[0].values.spec.components.ingressGateways[0].enabled = true" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.istio.overrides[0].values.spec.components.ingressGateways[0].name = \"istio-ingressgateway\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.istio.overrides[0].values.spec.components.ingressGateways[0].k8s.serviceAnnotations.\"service.beta.kubernetes.io/oci-load-balancer-internal\" = \"true\"" ${INSTALL_CONFIG_TO_EDIT}
fi

cat ${INSTALL_CONFIG_TO_EDIT}