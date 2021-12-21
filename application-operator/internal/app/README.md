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

### Demo instructions

- Deploy Verrazzano from master.
  
- To install the MetricsTemplate CRD, The MetricsTemplate resource, and the scrape generator mutating webhook, run this script.
  - `./scripts/install-app-resources.sh`
  
- Restart the application-operator pod to register the webhook. 
  (Otherwise, you will not be able to create deployments).
  - `kubectl delete pod -n verrazzano-system application-operator-XX-XX`
  
- Create the example namespace that contains these labels to enable Verrazzano and Istio.
  - `kubectl create namespace hello-helidon-namespace`
  - `kubectl label namespace hello-helidon-namespace verrazzano-managed=true istio-injection=enabled`
  
- Apply the example Deployment and Service to trigger the Metrics Template Webhook and Controller.
  - `kubectl apply -f resources/hello-helidon-test-deployment.yaml`

- Once the Deployment and Service are running, check the Prometheus targets for this title.
  - `hello-helidon-namespace_hello-helidon-deployment_<deploument-UID>`
  
### Demo Details
  
- Deployments have to have the `spec.selector.matchlabels` and `spec.template.metadata.labels` with the same custom value.
  
    ```yaml
    spec:
      selector:
        matchLabels:
          app: hello-helidon-application
      template:
        metadata:
          labels:
            app: hello-helidon-application
    ```
  - The current implementation looks for these labels to be present when applying the template.
    
- The LoadBalancer service for the application opens port `8080` to allow access to the pod from Prometheus.

- The endpoint `https://<pod-url>:8080/metrics` should successfully communicate metrics to Prometheus.
    
## Testing
The acceptance tests for this feature are disabled by default.
They can be executed using the following steps.

0. scripts/install-app-resources.sh
0. cd ../../../tests/e2e
0. KUBECONFIG=~/.kube/config ginkgo scrapegenerator/... -tags metrics_template_test
