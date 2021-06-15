#!/bin/bash

set -u

usage() {
  echo """
usage:

$0  -p <repo-name-pattern> -c <compartment-id> [-r <region> -f -w]

-r Region name
-p Repository pattern
-c Compartment ID
-f Force (no prompt)
-w Wait for delete
  """
}
REGION=us-phoenix-1
WAIT_FOR_STATE=""
FORCE=""

while getopts ":fwc:r:p:" opt; do
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
    w ) # wait for deletion
      WAIT_FOR_STATE="--wait-for-state DELETED"
      ;;
    \? )
      usage
      ;;
  esac
done
shift $((OPTIND -1))

if [ -z "${MATCH_PATTERN}" ]; then
  echo "Repository pattern not provided"
fi
if [ -z "${COMPARTMENT_ID}" ]; then
  echo "Compartment ID not provided"
fi

repo_ids="$(oci --region ${REGION} artifacts container repository list --compartment-id  ${COMPARTMENT_ID} | \
  jq -r --arg pattern "${MATCH_PATTERN}" '.data.items[] | select(."display-name"|test($pattern)) | .id')"

for id in ${repo_ids}; do
  repo_name=$(oci --region ${REGION} artifacts container repository get --repository-id ${id} | jq -r '.data."display-name"')
  echo "Deleting repository ${repo_name}, id: ${id}"
  oci --region ${REGION} artifacts container repository delete --repository-id ${id} ${FORCE} ${WAIT_FOR_STATE}
done