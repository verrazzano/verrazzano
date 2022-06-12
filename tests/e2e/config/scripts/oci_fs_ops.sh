#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

function usage {
    echo
    echo "usage: $0 [OPTIONS]"
    echo "  -o operation               Operation to perform - 'create' or 'delete'."
    echo "  -c compartment_ocid        Compartment OCID for creating the file system."
    echo "  -a ad_name                 Availability domain name for the file system."
    echo "  -n fs_name                 File system name."
    echo "  -m mt_name                 Mount name for the file system."
    echo "  -s subnet_ocid             Subnet OCID for creating the mount target."
    echo "  -p export_path             Export path for the mount target."
    echo "  -h                         Display this help message."
    echo
    exit 1
}


OPERATION=""
COMPARTMENT_OCID=""
AVAILABILITY_DOMAIN=""
FS_NAME=""
MT_NAME=""
SUBNET_OCID=""
EXPORT_PATH=""

function log () {
    echo "$(date '+[%Y-%m-%d %I:%M:%S %p]') : $1"
}

function createFileSystem() {
    log "Creating a file system: $FS_NAME in the availability domain: $AVAILABILITY_DOMAIN"
    oci fs file-system create --compartment-id $COMPARTMENT_OCID --availability-domain $AVAILABILITY_DOMAIN  --display-name $FS_NAME --wait-for-state ACTIVE
    if [ $? -ne 0 ]; then 
        log "Failed to create the file system: $FS_NAME"
        exit 1
    fi
    FS_OCID=$(oci fs file-system list --compartment-id $COMPARTMENT_OCID --availability-domain $AVAILABILITY_DOMAIN --display-name $FS_NAME --lifecycle-state ACTIVE | jq -r '.data[0].id')
    log "Successfully created the file system: $FS_NAME"
    log "File system OCID: $FS_OCID"
}

function createMountTarget() {
    log "Creating a mount target: $MT_NAME in the availability domain: $AVAILABILITY_DOMAIN"
    oci fs mount-target create --compartment-id $COMPARTMENT_OCID --availability-domain $AVAILABILITY_DOMAIN --subnet-id $SUBNET_OCID --display-name $MT_NAME --wait-for-state ACTIVE
    if [ $? -ne 0 ]; then 
        log "Failed to create the mount target: $MT_NAME"
        deleteFileSystem
        exit 1
    fi
    MT_OCID=$(oci fs mount-target list --compartment-id $COMPARTMENT_OCID --availability-domain $AVAILABILITY_DOMAIN --display-name $MT_NAME --lifecycle-state ACTIVE | jq -r '.data[0].id')
    log "Successfully created the mount target: $MT_NAME"
    log "File system OCID: $MT_OCID"
}

function createExport() {
    log "Creating a export: $EXPORT_PATH"
    FS_OCID=$(oci fs file-system list --compartment-id $COMPARTMENT_OCID --availability-domain $AVAILABILITY_DOMAIN --display-name $FS_NAME --lifecycle-state ACTIVE | jq -r '.data[0].id')
    EX_SET_OCID=$(oci fs export-set list --compartment-id $COMPARTMENT_OCID --availability-domain $AVAILABILITY_DOMAIN --display-name "$MT_NAME - export set" --lifecycle-state ACTIVE | jq -r '.data[0].id')
    oci fs export create --export-set-id $EX_SET_OCID --file-system-id $FS_OCID --path $EXPORT_PATH --wait-for-state ACTIVE
    if [ $? -ne 0 ]; then 
        log "Failed to create the export: $EXPORT_PATH"
        deleteMountTarget
        deleteFileSystem
        exit 1
    fi
    EX_OCID=$(oci fs export list --compartment-id $COMPARTMENT_OCID --export-set-id $EX_SET_OCID --file-system-id $FS_OCID --lifecycle-state ACTIVE | jq -r '.data[0].id')
    log "Successfully created the export: $EXPORT_PATH"
    log "Export OCID: $EX_OCID"
}

function deleteFileSystem() {
    log "Deleting the file system: $FS_NAME"
    FS_OCID=$(oci fs file-system list --compartment-id $COMPARTMENT_OCID --availability-domain $AVAILABILITY_DOMAIN --display-name $FS_NAME --lifecycle-state ACTIVE | jq -r '.data[0].id')
    if [ $? -ne 0 ]; then
        log "Error while fetching file system: $FS_NAME"
        exit 1
    fi
    log "File system OCID: $FS_OCID"
    oci fs file-system delete --file-system-id $FS_OCID --force --wait-for-state DELETED
    if [ $? -ne 0 ]; then
        log "Error while deleting the file system: $FS_NAME"
        exit 1
    fi
    log "Successfully deleted the file system: $FS_NAME"
}

function deleteMountTarget() {
    log "Deleting the mount target: $MT_NAME"
    MT_OCID=$(oci fs mount-target list --compartment-id $COMPARTMENT_OCID --availability-domain $AVAILABILITY_DOMAIN --display-name $MT_NAME --lifecycle-state ACTIVE | jq -r '.data[0].id')
    if [ $? -ne 0 ]; then
        log "Error while fetching the mount target: $MT_NAME"
        exit 1
    fi
    log "Mount target OCID: $MT_OCID"
    oci fs mount-target delete --mount-target-id $MT_OCID --force --wait-for-state DELETED
    if [ $? -ne 0 ]; then
        log "Error while deleting the mount target: $MT_NAME"
        exit 1
    fi
    log "Successfully deleted the mount target: $MT_NAME"
}

while getopts o:c:a:n:m:s:p:h flag
do
    case "$flag" in
        o) OPERATION=$OPTARG;;
        c) COMPARTMENT_OCID=$OPTARG;;
        a) AVAILABILITY_DOMAIN=$OPTARG;;
        n) FS_NAME=$OPTARG;;
        m) MT_NAME=$OPTARG;;
        s) SUBNET_OCID=$OPTARG;;
        p) EXPORT_PATH=$OPTARG;;
        h) usage;;
        *) usage;;
    esac
done

if [ -z "$OPERATION" ] ; then
    log "Operation must be specified"
    exit 1
fi
if [ -z "$COMPARTMENT_OCID" ] ; then
    log "Compartment OCID must be specified"
    exit 1
fi
if [ -z "$AVAILABILITY_DOMAIN" ] ; then
    log "Availability domain name must be specified"
    exit 1
fi
if [ -z "$FS_NAME" ] ; then
    log "File system name must be specified"
    exit 1
fi
if [ -z "$MT_NAME" ] ; then
    log "Mount target name must be specified"
    exit 1
fi
if [ $OPERATION == "create" ]; then
    if [ -z "$SUBNET_OCID" ] ; then
        log "Subnet OCID for mount target must be specified"
        exit 1
    fi
    if [ -z "$EXPORT_PATH" ] ; then
        log "Export path for the mount target must be specified"
        exit 1
    fi
fi

set -o pipefail
if [ $OPERATION == "create" ]; then
    createFileSystem
    createMountTarget
    createExport
elif [ $OPERATION == "delete" ]; then
    deleteMountTarget
    deleteFileSystem
else
    log "Invalid operation value: $OPERATION"
    usage
    exit 1
fi
