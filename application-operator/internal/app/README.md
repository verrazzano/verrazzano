# Verrazzano Application Extensions

This directory contains CRDs and templates related to Verrazzano application extensions.
These extensions are not yet installed by default with the product.

Note: Paths are relative to this README.md's directory.

## Installation
The script scripts/install-app-resources.sh installs the CRDs and manifests required for this feature.
These are required for this feature to function.
Once these CRDs and manifests are installed the Verrazzano Application Operator must be restarted.

## Testing
The acceptance tests for this feature are disabled by default.
They can be executed using the following steps.

0. scripts/install-app-resources.sh
0. cd ../../../tests/e2e
0. KUBECONFIG=~/.kube/config ginkgo scrapegenerator/... -tags metrics_template_test
