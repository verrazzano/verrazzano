#!/bin/bash
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -u

usage() {
  echo """
Delete OCIR repos matching a specific pattern in the specified tenancy compartment.  You can provide either the
full region name (e.g., "us-phoenix-1", or the region short name (e.g., "phx").

See https://docs.oracle.com/en-us/iaas/Content/Registry/Tasks/registrycreatingarepository.htm for details on
repository creation in an OCI tenancy.

Usage:

$0  -p <repo-name-pattern> -c <compartment-id> [-r <region> -f -w]

-r Region name (e.g., "us-phoenix-1")
-s Region short name (e.g., "phx", "lhr")
-p Repository pattern
-c Compartment ID
-f Force (no prompt)
-w Wait for delete
  """
}

REGION=
WAIT_FOR_STATE=""
FORCE=""
REGION_SHORT_NAME=""

while getopts ":fws:c:r:p:" opt; do
  case ${opt} in
    c ) # delete the backend
      COMPARTMENT_ID=${OPTARG}
      ;;
    r ) # drain the backend
      REGION=${OPTARG}
      ;;
    f ) # force delete, no prompt
      FORCE="--force"
      ;;
    p )
      MATCH_PATTERN="${OPTARG}"
      ;;
    s )
      REGION_SHORT_NAME="${OPTARG}"
      ;;
    w ) # wait for deletion
      WAIT_FOR_STATE="--wait-for-state DELETED"
      ;;
    \? )
      usage
      ;;
  esac
done
shift $((OPTIND -1))

if [ -z "${REGION}" ]; then
  if [ -z "${REGION_SHORT_NAME}" ]; then
    echo "Must provide either the full or the short region name"
    usage
    exit 1
  fi
  REGION=$(oci --region us-phoenix-1 iam region list | jq -r  --arg regionAbbr ${REGION_SHORT_NAME} '.data[] | select(.key|test($regionAbbr;"i")) | .name')
  if [ -z "${REGION}" ] || [ "null" == "${REGION}" ]; then
    echo "Invalid short region name ${REGION_SHORT_NAME}"
    usage
    exit 1
  fi
fi

if [ -z "${MATCH_PATTERN}" ]; then
  echo "Repository pattern not provided"
  usage
  exit 1
fi
if [ -z "${COMPARTMENT_ID}" ]; then
  echo "Compartment ID not provided"
  usage
  exit 1
fi

repo_ids="$(oci --region ${REGION} artifacts container repository list --compartment-id  ${COMPARTMENT_ID} | \
  jq -r --arg pattern "${MATCH_PATTERN}" '.data.items[] | select(."display-name"|test($pattern)) | .id')"

for id in ${repo_ids}; do
  repo_name=$(oci --region ${REGION} artifacts container repository get --repository-id ${id} | jq -r '.data."display-name"')
  echo "Deleting repository ${repo_name}, id: ${id}"
  oci --region ${REGION} artifacts container repository delete --repository-id ${id} ${FORCE} ${WAIT_FOR_STATE}
done
