#!/bin/bash
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Create the MetricsTemplate CRD
kubectl apply -f ${SCRIPT_DIR}/../crds/app.verrazzano.io_metricstemplates.yaml

# Create the MetricsBinding CRD
kubectl apply -f ${SCRIPT_DIR}/../crds/app.verrazzano.io_metricsbindings.yaml

# Create the MetricsTemplate resource
kubectl apply -f ${SCRIPT_DIR}/../resources/metrics-template-deployment.yaml

# Create the MutatingWebhookConfiguration object.  This object will eventually be
# moved to platform-operator/helm_config/charts/verrazzano-application-operator/templates/verrazzano-application-operator.yaml
kubectl apply -f - <<-EOF
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: verrazzano-application-scrape-generator
  namespace: verrazzano-system
  labels:
    app: verrazzano-application-operator
webhooks:
  - name: verrazzano-application-scrape-generator.verrazzano.io
    namespaceSelector:
      matchExpressions:
        - key: kubernetes.io/metadata.name
          operator: NotIn
          values: ["kube-system", "verrazzano-mc"]
        - key: verrazzano.io/namespace
          operator: NotIn
          values: ["verrazzano-system"]
    clientConfig:
      service:
        name: verrazzano-application-operator
        namespace: verrazzano-system
        path: "/scrape-generator"
    rules:
      - operations: ["CREATE","UPDATE"]
        apiGroups: ["*"]
        apiVersions: ["*"]
        resources: ["deployments","coherences","domains","pods","replicasets","statefulsets"]
        scope: "Namespaced"
    sideEffects: None
    failurePolicy: Fail
    matchPolicy: Equivalent
    timeoutSeconds: 30
    admissionReviewVersions:
      - v1beta1
      - v1
EOF

