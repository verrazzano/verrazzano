#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Script to mount OCI storage to ocne cluster
set -x
INSTANCE_IP="$1" 
API_SERVER_IP="$2"
OCI_MOUNT_IP="$3"
PREFIX="$4"
PRIVATE_KEY_PATH="$5"
OCI_EXPORT_PATH="$6"

ssh -o StrictHostKeyChecking=no opc@$INSTANCE_IP -i $PRIVATE_KEY_PATH "
    sudo yum install -y nfs-utils
    sudo mkdir -p /mnt/$OCI_EXPORT_PATH
    sudo mount $OCI_MOUNT_IP:/$OCI_EXPORT_PATH /mnt/$OCI_EXPORT_PATH
    for x in {0001..0020}; do
        sudo mkdir -p /mnt/olcne-master-filesystem/pv\$x && sudo chmod 777 /mnt/olcne-master-filesystem/pv\$x
    done
    ls /mnt/olcne-master-filesystem
"

cat << EOF | kubectl apply -f -
    apiVersion: storage.k8s.io/v1
    kind: StorageClass
    metadata:
        name: $PREFIX-nfs
        annotations:
            storageclass.kubernetes.io/is-default-class: "true"
    provisioner: kubernetes.io/no-provisioner
    volumeBindingMode: WaitForFirstConsumer
EOF

for n in {0001..0020}; do 
cat << EOF | kubectl apply -f -
    apiVersion: v1
    kind: PersistentVolume
    metadata:
        name: $PREFIX-pv$n
    spec:
        storageClassName: $PREFIX-nfs
        accessModes:
            - ReadWriteOnce
            - ReadWriteMany
        capacity:
            storage: 50Gi
        nfs:
            path: /$OCI_EXPORT_PATH/pv$n
            server: $OCI_MOUNT_IP
        volumeMode: Filesystem
        persistentVolumeReclaimPolicy: Recycle
EOF
done
