#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

OPERATOR_DIR=$1
VERRAZZANO_OPERATOR_IMAGE=$2
IMAGE_PULL_SECRET=${3:-verrazzano-container-registry}
GHCR_REPO=${4:-ghcr.io}
OUTPUT_FILE=${5:-/tmp/operator.yaml}


cat ${OPERATOR_DIR}/config/deploy/verrazzano-platform-operator.yaml | awk 'NR==1,/  namespace: verrazzano-install/{sub("  namespace: verrazzano-install", "  namespace: verrazzano-install\nimagePullSecrets:\n- name: '"${IMAGE_PULL_SECRET}"'")} 1' | \
sed -e "/VZ_INSTALL_IMAGE/a\              value: ${GHCR_REPO}/verrazzano/${VERRAZZANO_OPERATOR_IMAGE}\n            - name: IMAGE_PULL_SECRET\n              value: ${IMAGE_PULL_SECRET}" \
-e "/name: VZ_INSTALL_IMAGE/{n;d}" \
-e "s|IMAGE_NAME|${GHCR_REPO}/verrazzano/${VERRAZZANO_OPERATOR_IMAGE}|g" > ${OUTPUT_FILE}

cat ${OPERATOR_DIR}/config/crd/bases/install.verrazzano.io_verrazzanos.yaml >> ${OUTPUT_FILE}
cat ${OPERATOR_DIR}/config/crd/bases/clusters.verrazzano.io_verrazzanomanagedclusters.yaml >> ${OUTPUT_FILE}

