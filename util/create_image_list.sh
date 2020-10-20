#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

source ${SCRIPT_DIR}/../install/common.sh

# get image list from cluster and persist to output file
kubectl get pods --all-namespaces -o jsonpath="{..image}" |\tr -s '[[:space:]]' '\n' |\sort |\uniq -c | grep verrazzano | cut -c 6- | grep / | cut -d/ -f2- > ${SCRIPT_DIR}/verrazzano_img_list.txt

# add the acme solver (short lived container image)
echo $CERT_MANAGER_SOLVER_IMAGE:$CERT_MANAGER_SOLVER_TAG | grep / | cut -d/ -f2- >> ${SCRIPT_DIR}/verrazzano_img_list.txt

# handle helidon hello world image name mapping
cd ${SCRIPT_DIR}
sed -i 's/example-hello-world-helidon/example-helidon-greet-app-v1/' ${SCRIPT_DIR}/verrazzano_img_list.txt
