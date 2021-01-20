# Hello Helidon

The Hello Helidon example is a Helidon-based service that returns a "hello world" response when invoked. The application is specified in terms of OAM component and application configuration YAML files, and then deployed by applying those files.

# Prerequisites

You will need to fulfill the following prerequisites prior to running the example:

1. Create a Kubernetes cluster. A single node OKE cluster using the VM Standard 1.4 shape is recommended.
1. Install [Verrazzano](https://github.com/verrazzano/verrazzano#install-verrazzano).
1. Install the Verrazzano Application Operator.

# Deploy and Run the Example Application

Run the `install-hello-world.sh` script to create all of the necessary resources, install the example application, and test it.
```
./install-hello-world.sh
```

## Detailed Steps Description
The Helidon example installation script does the following:
1. Creates a namespace for the example.
    ```
    kubectl create namespace oam-hello-helidon
    ```
1. Applies the example OAM component and application.
    ```
    kubectl apply -f hello-helidon/
    ```
    The component is a `ContainerizedWorkload` that specifies the image and a port to expose.
    ```
    ...
      - name: hello-helidon-container
        image: "ghcr.io/verrazzano/example-helidon-greet-app-v1:0.1.10-3-20201016220428-56fb4d4"
        ports:
          - containerPort: 8080
            name: http
    ```
    The application references the single component and declares an `IngressTrait`.

# Access the Example Application
1. Get the public IP address of the Istio ingress gateway.
    ```
    kubectl get service -n "istio-system" "istio-ingressgateway" \
    -o jsonpath={.status.loadBalancer.ingress[0].ip}
    ```
1. Use `curl` to make a request to the service.
    ```
    curl -H "Host: hello-helidon.example.com" http://<public ip address>/greet
    ```

# Uninstall the Example
In order to uninstall the example, run:
```
./uninstall-hello-world.sh
```
