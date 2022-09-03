#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# This script edits the Verrazzano CR file in yaml format to enable components that are disabled by default,
# but required by the multi cluster test pipeline.
set -x
CLUSTER_COUNT=${1}
VZ_CR_FILE=${2}

yq -i eval '.spec.components.prometheusAdapter.enabled = true' "${VZ_CR_FILE}"
yq -i eval '.spec.components.kubeStateMetrics.enabled = true' "${VZ_CR_FILE}"
yq -i eval '.spec.components.prometheusPushgateway.enabled = true' "${VZ_CR_FILE}"
yq -i eval '.spec.components.jaegerOperator.enabled = true' "${VZ_CR_FILE}"
# For managed clusters, enable Jaeger operator and update the istio tracing configuration
if [ "${CLUSTER_COUNT}" -gt 1 ] ; then
    yq -i eval '.spec.components.istio.istioInstallArgs[0].name = "meshConfig.enableTracing"' "${VZ_CR_FILE}"
    yq -i eval '.spec.components.istio.istioInstallArgs[0].value = "true"' "${VZ_CR_FILE}"
    yq -i eval '.spec.components.istio.istioInstallArgs[1].name = "meshConfig.defaultConfig.tracing.sampling"' "${VZ_CR_FILE}"
    yq -i eval '.spec.components.istio.istioInstallArgs[1].value = "90.0"' "${VZ_CR_FILE}"
    yq -i eval '.spec.components.istio.istioInstallArgs[2].name = "meshConfig.defaultConfig.tracing.zipkin.address"' "${VZ_CR_FILE}"
    yq -i eval '.spec.components.istio.istioInstallArgs[2].value = "jaeger-verrazzano-managed-cluster-collector.verrazzano-monitoring:9411"' "${VZ_CR_FILE}"
    yq -i eval '.spec.components.istio.istioInstallArgs[3].name = "meshConfig.defaultConfig.tracing.tlsSettings.mode"' "${VZ_CR_FILE}"
    yq -i eval '.spec.components.istio.istioInstallArgs[3].value = "ISTIO_MUTUAL' "${VZ_CR_FILE}"
fi

echo "VZ CR to be applied:"
cat "${VZ_CR_FILE}"
