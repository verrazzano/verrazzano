#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Usage: ./process_olcne_install_yaml.sh <CONFIG_FILE_TO_EDIT> [ENVIRONMENT_NAME INSTALL_PROFILE]
# - Updates the given yaml file

INSTALL_CONFIG_TO_EDIT=$1
ENVIRONMENT_NAME=${2:-"default"}
INSTALL_PROFILE=${3:-"dev"}

yq -i eval ".spec.environmentName = \"$ENVIRONMENT_NAME\"" $INSTALL_CONFIG_TO_EDIT
yq -i eval ".spec.profile = \"$INSTALL_PROFILE\"" $INSTALL_CONFIG_TO_EDIT
yq -i eval ".spec.profile = \"$INSTALL_PROFILE\"" $INSTALL_CONFIG_TO_EDIT
yq -i eval ".spec.ingress.nginxInstallArgs |= [{\"name\": \"controller.service.annotations.service\\.beta\\.kubernetes\\.io/oci-load-balancer-shape\", \"value\": \"10Mbps\"}]" $INSTALL_CONFIG_TO_EDIT
yq -i eval ".spec.ingress.nginxInstallArgs += [{\"name\": \"controller.service.annotations.service\\.beta\\.kubernetes\\.io/oci-load-balancer-list-management-mode\", \"value\": \"None\"}]" $INSTALL_CONFIG_TO_EDIT
yq -i eval ".spec.istio.istioInstallArgs |= [{\"name\": \"gateways.istio-ingressgateway.serviceAnnotations.service\\.beta\\.kubernetes\\.io/oci-load-balancer-shape\", \"value\": \"10Mbps\"}]" $INSTALL_CONFIG_TO_EDIT
yq -i eval ".spec.istio.istioInstallArgs += [{\"name\": \"gateways.istio-ingressgateway.serviceAnnotations.service\\.beta\\.kubernetes\\.io/oci-load-balancer-list-management-mode\", \"value\": \"None\"}]" $INSTALL_CONFIG_TO_EDIT
sed 's/controller.service.annotations./controller.service.annotations."/g' $INSTALL_CONFIG_TO_EDIT > /tmp/vz.yaml
mv /tmp/vz.yaml $INSTALL_CONFIG_TO_EDIT
sed 's/gateways.istio-ingressgateway.serviceAnnotations./gateways.istio-ingressgateway.serviceAnnotations."/g' $INSTALL_CONFIG_TO_EDIT > /tmp/vz.yaml
mv /tmp/vz.yaml $INSTALL_CONFIG_TO_EDIT
sed 's/oci-load-balancer-shape/oci-load-balancer-shape"/g' $INSTALL_CONFIG_TO_EDIT > /tmp/vz.yaml
mv /tmp/vz.yaml $INSTALL_CONFIG_TO_EDIT
sed 's/oci-load-balancer-list-management-mode/oci-load-balancer-list-management-mode"/g' $INSTALL_CONFIG_TO_EDIT > /tmp/vz.yaml
mv /tmp/vz.yaml $INSTALL_CONFIG_TO_EDIT
