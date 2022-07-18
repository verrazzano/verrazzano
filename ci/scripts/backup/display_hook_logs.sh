#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -o pipefail

echo "Fetching backup hook logs ..."
hookLogFile=$(kubectl exec -it -n verrazzano-system  vmi-system-es-master-0 -- ls -alt --time=ctime /tmp/ | grep verrazzano | cut -d ' ' -f9 | head -1)
hookLog=$(kubectl exec -it -n verrazzano-system  vmi-system-es-master-0 -- cat /tmp/${hookLogFile})
echo $hookLog