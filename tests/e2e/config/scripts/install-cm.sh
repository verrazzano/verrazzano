# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Installs Cert-Manager, for simulating an customer-managed Cert-Manager instance
#
# Arguments:
# 1 - certManagerNamespace to install Cert-Manager, defaults to "cert-manager"
# 2 - certManagerNamespace to use for the clusterResourceNamespace, defaults to the certManagerNamespace arg
#
# Optional:
# CERTMANAGER_VERSION - Cert-Manager version to use, defaults to v1.11.0
# DRY_RUN - Runs Helm in dry-run mode
#

certManagerNamespace=${1:-cert-manager}
clusterResourceNamespace=${2:-${certManagerNamespace}}

CERTMANAGER_VERSION=${CERTMANAGER_VERSION:-"v1.11.0"}
DRY_RUN=${DRY_RUN:-""}

echo installing Cert-Manager in certManagerNamespace ${certManagerNamespace}, clusterResourceNamespace=${clusterResourceNamespace}

kubectl create ns ${clusterResourceNamespace} || true

set -x
helm upgrade --install ${DRY_RUN} \
  cert-manager jetstack/cert-manager \
  --certManagerNamespace ${certManagerNamespace} \
  --create-certManagerNamespace \
  --set clusterResourceNamespace=${clusterResourceNamespace} \
  --version "${CERTMANAGER_VERSION}" \
  --set installCRDs=true
