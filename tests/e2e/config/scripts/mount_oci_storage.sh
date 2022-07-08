#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Script to mount OCI storage to ocne cluster
set -x
API_SERVER_IP="$1"
CONTROL_PLANE_IP="$2"
WORKER_IP="$3"
OCI_MOUNT_IP="$4"
PREFIX="$5"
PRIVATE_KEY_PATH="$6"
OCI_EXPORT_PATH="$7"

ssh -o StrictHostKeyChecking=no opc@$WORKER_IP -i $PRIVATE_KEY_PATH "
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

cat << EOF | kubectl apply -f -
    apiVersion: v1
    kind: ConfigMap
    metadata:
        name: recycler-pod-config
        namespace: kube-system
    data:
        recycler-pod.yaml: |
            apiVersion: v1
            kind: Pod
            metadata:
            name: pv-recycler
            namespace: default
            spec:
            restartPolicy: Never
            volumes:
            - name: vol
                hostPath:
                path: /tmp
            containers:
            - name: pv-recycler
                image: "busybox"
                command: ["/bin/sh", "-c", "test -e /scrub && rm -rf /scrub/..?* /scrub/.[!.]* /scrub/*  && test -z \"$(ls -A /scrub)\" || exit 1"]
                volumeMounts:
                - name: vol
                mountPath: /scrub
EOF

ssh -o StrictHostKeyChecking=no opc@$CONTROL_PLANE_IP -i $PRIVATE_KEY_PATH '
    K8S_CONTROLLER_MANAGER_PATH=/etc/kubernetes/manifests/kube-controller-manager.yaml
    sudo yq -i eval \'.spec.containers[0].command += "--pv-recycler-pod-template-filepath-nfs=/etc/recycler-pod.yaml"\' "$K8S_CONTROLLER_MANAGER_PATH"
    sudo yq -i eval \'.spec.containers[0].volumeMounts += [{"name": "recycler-config-volume", "mountPath": "/etc/recycler-pod.yaml", "subPath": "recycler-pod.yaml"}]\' "$K8S_CONTROLLER_MANAGER_PATH"
    sudo yq -i eval \'.spec.volumes += [{"name": "recycler-config-volume", "configMap": {"name": "recycler-pod-config"}}]\' "$K8S_CONTROLLER_MANAGER_PATH"
'
