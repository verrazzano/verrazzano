
# Hello World Helidon Demo Application

This demo provides a simple *Hello World* REST service running on [Helidon](https://helidon.io)

###Install demo

* Pre-requisites: Install Verrazzano following the [installation instructions](../install/README.md).

* Install demo
```
kubectl apply -f ./hello-world-model.yaml
kubectl apply -f ./hello-world-binding.yaml
```
* Verify if all objects have started:
```
kubectl get all -n greet
```
* Get the External IP for istio-ingressgateway service
```
kubectl get service istio-ingressgateway -n istio-system
```
* Use the external IP to call the different endpoints of the greeting REST service:
    - Default greeting message: `curl -X GET http://<external_ip>/greet`
    - Greet Robert: `curl -X GET http://<external_ip>/greet/Robert`

###Uninstall Demo

* Uninstall demo
```
kubectl delete -f ./hello-world-binding.yaml
kubectl delete -f ./hello-world-model.yaml
```
