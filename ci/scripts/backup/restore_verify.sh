#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -x
set -o pipefail

if [ -z "$BACKUP_ID" ]; then
  echo "This script must only be called from Jenkins and requires 'BACKUP_ID' environment variable to be set"
  exit 1
fi

ES_URL=$(kubectl get vz -o jsonpath={.items[].status.instance.elasticUrl})
VZ_PASSWORD=$(kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode)

REQUEST_JSON_BODY=/tmp/input.json
cat <<EOF >> ${REQUEST_JSON_BODY}
      {
        "query": {
          "terms": {
            "_id": ["${BACKUP_ID}"]
          }
        }
      }
EOF

cat /tmp/input.json

CHECK_BACKUP_ID=$(curl -ks -H "Content-Type: application/json" "${ES_URL}/verrazzano-system/_search?" -u verrazzano:${VZ_PASSWORD} -d @${REQUEST_JSON_BODY} | jq -r '.hits.hits[0]._id')

rm -rf /tmp/input.json

if [ ${CHECK_BACKUP_ID} == ${BACKUP_ID} ]; then
  echo "True"
  exit 0
fi
echo "False"
exit 1











