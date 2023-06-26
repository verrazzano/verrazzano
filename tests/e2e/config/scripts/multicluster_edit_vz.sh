#!/bin/bash
#
# Copyright (c) 2022, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# This script edits the Verrazzano CR file in yaml format to enable components that are disabled by default,
# but required by the multi cluster test pipeline.
set -x
CLUSTER_COUNT=${1}
VZ_CR_FILE=${2}
MINIMAL_INSTALL=${3}

if [ -n "${MINIMAL_INSTALL}" ] && [ "${MINIMAL_INSTALL}" = "true" ] ; then
  echo "Minimal install, not applying the overrides"
  exit 0
fi

yq -i eval '.spec.components.prometheusAdapter.enabled = true' "${VZ_CR_FILE}"
yq -i eval '.spec.components.kubeStateMetrics.enabled = true' "${VZ_CR_FILE}"
yq -i eval '.spec.components.prometheusPushgateway.enabled = true' "${VZ_CR_FILE}"
yq -i eval '.spec.components.jaegerOperator.enabled = true' "${VZ_CR_FILE}"
# For managed clusters, enable Jaeger operator and update the istio tracing configuration
if [ "${CLUSTER_COUNT}" -gt 1  ]; then
  yq -i eval '.spec.components.istio.overrides.[0].values.apiVersion = "install.istio.io/v1alpha1"' ${VZ_CR_FILE}
  yq -i eval '.spec.components.istio.overrides.[0].values.kind = "IstioOperator"' ${VZ_CR_FILE}
  yq -i eval '.spec.components.istio.overrides.[0].values.spec.meshConfig.defaultConfig.tracing.sampling = 90' ${VZ_CR_FILE}
  yq -i eval '.spec.components.istio.overrides.[0].values.spec.meshConfig.defaultConfig.tracing.zipkin.address = "jaeger-verrazzano-managed-cluster-collector.verrazzano-monitoring.svc.cluster.local.:9411"' ${VZ_CR_FILE}
  yq -i eval '.spec.components.istio.overrides.[0].values.spec.meshConfig.enableTracing = true' ${VZ_CR_FILE}
fi
#For admin cluster only, enable Argo CD
if [ "${CLUSTER_COUNT}" -eq 1  ]; then
  yq -i eval '.spec.components.argoCD.enabled = true' "${VZ_CR_FILE}"
fi
echo "VZ CR to be applied:"
cat "${VZ_CR_FILE}"
