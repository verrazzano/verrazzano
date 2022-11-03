#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

function usage {
    echo
    echo "usage: $0 [-o operation] [-c compartment_ocid] [-s subdomain_name] "
    echo "  -o  operation               'create' or 'delete'. Optional.  Defaults to 'create'."
    echo "  -c  compartment_ocid        Compartment OCID. Optional.  Defaults to TIBURON-DEV compartment OCID."
    echo "  -s  subdomain_name         subdomain prefix for v8o.io. Required."
    echo "  -k DNS scope                Specifies  to operate only on resources that have a matching DNS scope.Optional. GLOBAL, PRIVATE"
    echo "  -h                         Help"
    echo
    exit 1
}

SUBDOMAIN_NAME=""
COMPARTMENT_OCID="${TF_VAR_compartment_id}"
OPERATION="create"
DNS_SCOPE="GLOBAL"
# view id for phx (default)
VCN_VIEW_ID="${VCN_VIEW_ID}"
log () {
  echo "$(date '+[%Y-%m-%d %I:%M:%S %p]') : $1"
}

while getopts o:c:s:k:h flag
do
    case "${flag}" in
        o) OPERATION=${OPTARG};;
        c) COMPARTMENT_OCID=${OPTARG};;
        s) SUBDOMAIN_NAME=${OPTARG};;
        k) DNS_SCOPE_INPUT=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

if [ -z "${SUBDOMAIN_NAME}" ] ; then
    echo "subdomain name must be set!"
    exit 1
fi

if [ ${TEST_ENV} != "kind_oci_dns" ] && [ ${DNS_SCOPE} == "PRIVATE" ]; then
  if [ ${V80_COMPARTMENT_OCID} == "" ]; then
      echo "Jenkins runner compartment ocid must be set!"
      exit 1
  fi
fi

if [ ${TEST_ENV} != "ocidns_oke" ] && [ ${TEST_ENV} != "kind_oci_dns" ]; then
  if [ ${DNS_SCOPE} == "PRIVATE" ];then
    echo "Invalid TEST_ENV for DNS_SCOPE=PRIVATE " ${TEST_ENV}
    exit 1
  fi
fi


if [ "${DNS_SCOPE_INPUT:-}" ] ; then
  if [ ${DNS_SCOPE_INPUT} == "GLOBAL" ] || [ ${DNS_SCOPE_INPUT} == "PRIVATE" ]; then
     DNS_SCOPE=${DNS_SCOPE_INPUT}
  fi
fi

set -o pipefail
if [ ${DNS_SCOPE} == "PRIVATE" ];then
  ZONE_NAME="${SUBDOMAIN_NAME}-private.v8o.io"
else
  ZONE_NAME="${SUBDOMAIN_NAME}.v8o.io"
fi

zone_ocid=""
status_code=1
if [ $OPERATION == "create" ]; then
  # the installation will require the "patch" command, so will install it now.  If it's already installed yum should
  # exit
  sudo yum -y install patch >/dev/null 2>&1

  if [ ${DNS_SCOPE} == "PRIVATE" ];then
    zone_ocid=$(oci dns zone create -c ${COMPARTMENT_OCID} --name ${ZONE_NAME} --zone-type PRIMARY --scope ${DNS_SCOPE} --view-id ${VCN_VIEW_ID}| jq -r ".data | .[\"id\"]"; exit ${PIPESTATUS[0]})
    status_code=$?
    if [ ${status_code} -ne 0 ]; then
      log "Failed creating private zone, attempting to fetch zone to see if it already exists"
      oci dns zone get --zone-name-or-id ${ZONE_NAME}
    fi

    if [ ${TEST_ENV} == "ocidns_oke" ]; then
      VCN_ID=$(oci network vcn list --compartment-id "${COMPARTMENT_OCID}" --display-name "${TF_VAR_label_prefix}-oke-vcn" | jq -r '.data[0].id')
    elif [ ${TEST_ENV} == "kind_oci_dns" ]; then
      VCN_ID=$(oci network vcn list --compartment-id "${V80_COMPARTMENT_OCID}" --display-name ${JENKINS_VCN} | jq -r '.data[0].id')
    fi
    if [ $? -ne 0 ];then
        log "Failed to fetch vcn '${TF_VAR_label_prefix}-oke-vcn'"
        exit 1
    fi

    DNS_RESOLVER_ID=$(oci network vcn-dns-resolver-association get --vcn-id ${VCN_ID} | jq '.data["dns-resolver-id"]' -r)
    DNS_UPDATE=$(oci dns resolver update --resolver-id ${DNS_RESOLVER_ID} --attached-views '[{"viewId":"'"${VCN_VIEW_ID}"'"}]' --scope PRIVATE --force)
    if [ $? -ne 0 ];then
        log "Failed to update vcn '${TF_VAR_label_prefix}-oke-vcn' with private view"
        exit 1
    fi
  else
    zone_ocid=$(oci dns zone create -c ${COMPARTMENT_OCID} --name ${ZONE_NAME} --zone-type PRIMARY --scope ${DNS_SCOPE} | jq -r ".data | .[\"id\"]"; exit ${PIPESTATUS[0]})
    status_code=$?
    if [ ${status_code} -ne 0 ]; then
      log "Failed creating public zone, attempting to fetch zone to see if it already exists"
      oci dns zone get --zone-name-or-id ${ZONE_NAME}
    fi
  fi

elif [ $OPERATION == "delete" ]; then
  DNS_ZONE_OCID=`(oci dns zone list --compartment-id ${COMPARTMENT_OCID} --scope ${DNS_SCOPE} --name ${ZONE_NAME} | jq -r '.data[].id')`
  oci dns zone delete --zone-name-or-id ${DNS_ZONE_OCID} --scope ${DNS_SCOPE} --force
  status_code=$?
  if [ ${status_code} -ne 0 ]; then
    log "DNS zone deletion failed on first try. Retrying once."
    oci dns zone delete --zone-name-or-id ${DNS_ZONE_OCID} --scope ${DNS_SCOPE} --force
    status_code=$?
  fi
else
  log "Unknown operation: ${OPERATION}"
  usage
fi

if [ ${status_code} -eq 0 ]; then
  # OCI CLI query succeeded
  if [ $OPERATION == "create" ]; then
    echo $zone_ocid
  else
    exit 0
  fi
else
  # OCI CLI generated an error exit code
  log "Error invoking OCI CLI to perform DNS zone operation"
  exit 1
fi
