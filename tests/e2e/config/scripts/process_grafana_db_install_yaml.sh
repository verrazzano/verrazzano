#!/bin/bash

# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

INSTALL_CONFIG_TO_EDIT=$1
echo "Editing install config file for Grafana DB ${INSTALL_CONFIG_TO_EDIT}"
  yq -i eval ".spec.components.grafana.database.host = \"mysql.verrazzano-install.svc.cluster.local\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.grafana.database.name = \"grafana\"" ${INSTALL_CONFIG_TO_EDIT}

cat ${INSTALL_CONFIG_TO_EDIT}
