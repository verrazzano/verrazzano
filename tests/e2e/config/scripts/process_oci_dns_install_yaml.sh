#!/bin/bash
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

INSTALL_CONFIG_TO_EDIT=$1
echo "Editing install config file for OCI DNS ${INSTALL_CONFIG_TO_EDIT}"
yq -i eval ".spec.environmentName = \"${VZ_ENVIRONMENT_NAME}\"" ${INSTALL_CONFIG_TO_EDIT}
yq -i eval ".spec.profile = \"${INSTALL_PROFILE}\"" ${INSTALL_CONFIG_TO_EDIT}
yq -i eval ".spec.components.dns.oci.dnsZoneCompartmentOCID = \"${OCI_DNS_COMPARTMENT_OCID}\"" ${INSTALL_CONFIG_TO_EDIT}
yq -i eval ".spec.components.dns.oci.dnsZoneOCID = \"${OCI_DNS_ZONE_OCID}\"" ${INSTALL_CONFIG_TO_EDIT}
yq -i eval ".spec.components.dns.oci.dnsZoneName = \"${OCI_DNS_ZONE_NAME}\"" ${INSTALL_CONFIG_TO_EDIT}
if [ $INSTALL_PROFILE == "dev" ]; then
  yq -i eval ".spec.components.keycloak.mysql.mysqlInstallArgs.[0].name = \"persistence.enabled\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.keycloak.mysql.mysqlInstallArgs.[0].value = \"false\"" ${INSTALL_CONFIG_TO_EDIT}
fi
cat ${INSTALL_CONFIG_TO_EDIT}
