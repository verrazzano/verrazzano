#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -o pipefail

if [ -z "$BACKUP_OPENSEARCH" ] ||  [ -z "$VELERO_NAMESPACE" ] || [ -z "$VELERO_SECRET_NAME" ]
   [ -z "$BACKUP_STORAGE" ] || [ -z "$OCI_OS_NAMESPACE" ] ; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

kubectl apply -f - <<EOF
    apiVersion: velero.io/v1
    kind: Backup
    metadata:
      name: ${BACKUP_OPENSEARCH}
      namespace: ${VELERO_NAMESPACE}
    spec:
      includedNamespaces:
        - verrazzano-system
      labelSelector:
        matchLabels:
          verrazzano-component: opensearch
      defaultVolumesToRestic: false
      storageLocation: ${BACKUP_STORAGE}
      hooks:
        resources:
          -
            name: ${BACKUP_RESOURCE}
            includedNamespaces:
              - verrazzano-system
            labelSelector:
              matchLabels:
                statefulset.kubernetes.io/pod-name: vmi-system-es-master-0
            post:
              -
                exec:
                  container: es-master
                  command:
                    - /usr/share/opensearch/bin/verrazzano-backup-hook
                    - -operation
                    - backup
                    - -velero-backup-name
                    - ${BACKUP_OPENSEARCH}
                  onError: Fail
                  timeout: 10m
EOF

kubectl get backup.velero.io -n ${VELERO_NAMESPACE} ${BACKUP_OPENSEARCH}


