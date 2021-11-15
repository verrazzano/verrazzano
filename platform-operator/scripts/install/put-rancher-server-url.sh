#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

RANCHER_ADMIN_PASSWORD="$1"
RANCHER_HOSTNAME="$2"

echo "Get Rancher access token."
get_rancher_access_token "${RANCHER_HOSTNAME}" "${RANCHER_ADMIN_PASSWORD}"
if [ $? -ne 0 ] ; then
  echo "Failed to get Rancher access token. Continuing without setting Rancher server URL."
  return 0
fi

if [ -z "${RANCHER_ACCESS_TOKEN}" ]; then
  echo "Failed to get valid Rancher access token. Continuing without setting Rancher server URL."
  return 0
fi
echo "Set Rancher server URL to https://${RANCHER_HOSTNAME}"
curl_args=("https://${RANCHER_HOSTNAME}:$(get_nginx_nodeport)/v3/settings/server-url" $(get_rancher_resolve ${RANCHER_HOSTNAME}) \
      -H 'content-type: application/json' \
      -H "Authorization: Bearer ${RANCHER_ACCESS_TOKEN}" \
      -X PUT \
      --data-binary '{"name":"server-url","value":"'https://${RANCHER_HOSTNAME}'"}' \
      --insecure)
call_curl 200 http_response http_status curl_args || true
if [ ${http_status:--1} -ne 200 ]; then
  echo "Failed to set Rancher server URL. Continuing without setting Rancher server URL."
  return 0
else
  echo "Successfully set Rancher server URL."
fi