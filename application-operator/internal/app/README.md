# Verrazzano Application Extensions

This directory contains CRDs and templates related to Verrazzano application extensions.
These extensions are not yet installed by default with the product.

Note: Paths are relative to this README.md's directory.

## Installation
The script scripts/install-app-resources.sh installs the CRDs and manifests required for this feature.
These are required for this feature to function.
Once these CRDs and manifests are installed the Verrazzano Application Operator must be restarted.

## Demo
Use the following steps to demonstrate the function of the metrics template scrape generator:

- Deploy a Verrazzano version with the updated controller and webhook
  
- run the script `scripts/install-app-resources.sh`
  to install the MetricsTemplate CRD, The MetricsTemplate resource, and the scrape generator mutating webhook.
  
- restart the application-operator pod to register the webhook. 
  (Otherwise, you will not be able to create deployments)
  
- Create a namespace that contains these labels to enable Verrazzano and Istio:
  
  ```kubectl label namespace <namespace-name> verrazzano-managed=true istio-injection=enabled```
  
- Create a Deployment in the labeled namespace. 
  - Make sure to populate the `spec.selector.matchlabels` and `spec.template.metadata.labels` with the same custom value:
  
    ```app: <application-name>```
    
- Create a LoadBalancer service with port `8080` to allow access to the pod from Prometheus
  
- For a sample application, you can use the Deployment and Service located in `resources/hello-helidon-test-deployment.yaml`

- Once the Deployment and Service are running, check the Prometheus targets for a target titled `<namespace>_<deployment-name>-<deploument-UID>`
  - For now, the container ports will show up in the target as unavailable. 
    Fixing this issue will require more elaboration and time and will be sorted out in the near future.
    
## Testing
The acceptance tests for this feature are disabled by default.
They can be executed using the following steps.

0. scripts/install-app-resources.sh
0. cd ../../../tests/e2e
0. KUBECONFIG=~/.kube/config ginkgo scrapegenerator/... -tags metrics_template_test
