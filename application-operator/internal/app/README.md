# Verrazzano Application Extensions

This directory contains CRDs and templates related to Verrazzano application extensions.
These extensions are not yet installed by default with the product.

Note: Paths are relative to this README.md's directory.

## Installation
The script scripts/install-app-resources.sh installs the CRDs and manifests required for this feature.
These are required for this feature to function.
Once these CRDs and manifests are installed the Verrazzano Application Operator must be restarted.

## Demo
Use the following steps to demonstrate the function of the metrics template scrape generator.

### Demo Instructions

1. Install Verrazzano from master.
  
2. Install the MetricsTemplate CRD, MetricsBinding CRD, default MetricsTemplate resource, and the metrics binding mutating webhooks, run this script.
   - `./scripts/install-app-resources.sh`
  
3. Restart the application-operator pod to register the webhooks. 
    - `kubectl delete pod -n verrazzano-system application-operator-XX-XX`
  
4. Create the example namespace that contains these labels to enable Verrazzano and Istio.
    - `kubectl create namespace hello-helidon-namespace`
    - `kubectl label namespace hello-helidon-namespace verrazzano-managed=true istio-injection=enabled`
  
5. Apply the example Deployment and Service to trigger the webhooks and Controller.
    - `kubectl apply -f resources/workloads/hello-helidon-deployment.yaml`

6. Once the Deployment and Service are running, check the Prometheus targets for this title.
    - `hello-helidon-namespace_hello-helidon-deployment_apps_v1_Deployment`
  
### Demo Details
  
- The LoadBalancer service for the application opens port `8080` to allow access to the pod from Prometheus.

- The endpoint `https://<pod-url>:8080/metrics` should successfully communicate metrics to Prometheus.
    
## Acceptance Tests
The acceptance tests for this feature are not runnable from Jenkins at this time and must be run locally.
To run the accceptance tests:

1. Run steps 1--3 from the Demo Instructions above
2. cd ../../../tests/e2e
3. ginkgo -v --no-color metricsbinding/...
