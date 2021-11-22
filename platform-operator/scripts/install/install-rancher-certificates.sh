#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

export VERRAZZNO_KUBECONFIG_DISABLED=true

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

log "Using ACME staging, create staging certs secret for Rancher"
acme_staging_certs=${TMP_DIR}/ca-additional.pem
echo -n "" > ${acme_staging_certs}
curl_args=(--output ${TMP_DIR}/int-r3.pem "https://letsencrypt.org/certs/staging/letsencrypt-stg-int-r3.pem")
call_curl 200 http_response http_status curl_args || true
if [ ${http_status:--1} -ne 200 ]; then
  log "Error downloading LetsEncrypt Staging intermediate R3 cert"
else
  cat ${TMP_DIR}/int-r3.pem >> ${acme_staging_certs}
fi
curl_args=(--output ${TMP_DIR}/int-e1.pem "https://letsencrypt.org/certs/staging/letsencrypt-stg-int-e1.pem")
call_curl 200 http_response http_status curl_args || true
if [ ${http_status:--1} -ne 200 ]; then
  log "Error downloading LetsEncrypt Staging intermediate E1 cert"
else
  cat ${TMP_DIR}/int-e1.pem >> ${acme_staging_certs}
fi
curl_args=(--output ${TMP_DIR}/root-x1.pem "https://letsencrypt.org/certs/staging/letsencrypt-stg-root-x1.pem")
call_curl 200 http_response http_status curl_args || true
if [ ${http_status:--1} -ne 200 ]; then
  log "Error downloading LetsEncrypt Staging X1 Root cert"
else
  cat ${TMP_DIR}/root-x1.pem >> ${acme_staging_certs}
fi
kubectl -n cattle-system create secret generic tls-ca-additional --from-file=ca-additional.pem=${acme_staging_certs}
