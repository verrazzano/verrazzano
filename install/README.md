# verrazzano-oke-install

Setup [Verrazzano](http://verrazzano.pages.oracledx.com/website/docs/docs/demo-apps/bobs-books//) on [Oracle Kubernetes Engine](https://docs.cloud.oracle.com/en-us/iaas/Content/ContEng/Concepts/contengoverview.htm) 

# Quick Start
```
export VERRAZZANO_KUBECONFIG=<OKE cluster designated for a Verrazzano installation>
kubectl create secret docker-registry ocr --docker-username=<username> --docker-password=<password> --docker-server=container-registry.oracle.com
./install-oke.sh
```

# System Configuration

## Resource Requirements

The following configuration has proven sufficient to install Verrazzano and deploy Bob's demo.

OCI Compute instance shape `VM.Standard2.4` which has:
* 4 `2.0 GHz Intel® Xeon® Platinum 8167M` cores
* 60 GB of memory
* Select disk size of at least 200 GB.  A minimum of 100 GB of storage required for docker images.

The following environments are not sufficient for running Verrazzano:
* Dual-core Mac with 16 GB of memory

## Software Requirements

The following software must be installed on your system.  
* Kubectl
* Helm
* jq
* openssl

## Steps to Install Prerequisites on Linux
This section contains the instructions for configuring a Linux system with the required software:
```text
# kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl

# helm
curl https://get.helm.sh/helm-v3.1.2-linux-amd64.tar.gz -o helm-v3.1.2-linux-amd64.tar.gz
tar xzvf helm-v3.1.2-linux-amd64.tar.gz
sudo mv linux-amd64/helm /usr/local/bin/helm
```

# General Instructions for Installing Dependencies

## Install kubectl
Follow the [kubectl install instructions](https://kubernetes.io/docs/tasks/tools/install-kubectl/) on the Kubernetes webstie to install kubectl CLI.

Verify that `kubectl` is installed:
```
kubectl version
```

## Install helm
Follow the [Helm install instructions](https://helm.sh/docs/intro/install/) on the helm webstie to install helm CLI.

Verify that `helm` is installed:
```
helm version
```

# Install in OKE Cluster using xip.io

export KUBECONFIG to point at the OKE cluster designated for a Verrazzano installation
```
export KUBECONFIG=/Users/myuser/.kube/myv8ocluster/config
```

## Install Istio
Before installing istio you must create the secret "ocr" in the default namespace.
The "ocr" secret is needed for pulling images from the container-registry.oracle.com repository.
```text
kubectl create secret docker-registry ocr --docker-username=<username> --docker-password=<password> --docker-server=container-registry.oracle.com
```
```text
./1-install-istio.sh
```

## Install Nginx, Cert-Manager and Rancher
```
./2-install-rancher.sh
```
The default installation uses [xip.io](http://xip.io/) for DNS. The default rancher URL uses the IP of the NGINX Ingress
```
kubectl get svc -n ingress-nginx
NAME                                               TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)                      AGE
ingress-controller-nginx-ingress-controller        LoadBalancer   10.96.75.235    150.136.20.133   80:31532/TCP,443:32015/TCP   43m
```

To verify that Rancher is installed, open https://rancher.default.150.136.20.133.xip.io

## Install Verrazzano

Run the following script
```
./3-install-verrazzano.sh
```

The administrator initial password:
```
User: verrazzano

Password:
kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath="{.data.password}" | base64 --decode; echo
```

## Install Keycloak

Run the following script
```
./4-install-keycloak.sh
```

# Install in OKE Cluster Using OCI DNS
Use this option if you would like to have OCI DNS records created for you automatically
during the installs of Verrazzano components.  For this option, you would not use the
**./2-install-rancher.sh** script but instead use **./2-install-rancher-oci-dns.sh**

export KUBECONFIG to point at the OKE cluster designated for a Verrazzano installation
```
export KUBECONFIG=/Users/myuser/.kube/myv8ocluster/config
```

## Install Istio
Before installing istio you must create the secret "ocr" in the default namespace.
The "ocr" secret is needed for pulling images from the container-registry.oracle.com repository.
```text
kubectl create secret docker-registry ocr --docker-username=<username> --docker-password=<password> --docker-server=container-registry.oracle.com
```
```text
./1-install-istio.sh
```

## Install Nginx, Cert-Manager, External-DNS and Rancher

**./2-install-rancher-oci-dns.** requires a number of environment variables to be set prior to running the script.  These environment
variables specify the OCI environment needed for creating OCI DNS records.

Environment Variable | Required | Description
--- | --- | --- |
`EMAIL_ADDRESS` | Yes | Email address
`OCI_COMPARTMENT_OCID` | Yes | OCI Compartment OCID
`OCI_DNS_ZONE_NAME` | Yes | Name of OCI DNS Zone
`OCI_DNS_ZONE_OCID` | Yes | OCI DNS Zone OCID
`OCI_FINGERPRINT` | Yes | OCI fingerprint
`OCI_PRIVATE_KEY_FILE` | Yes | OCI private key file
`OCI_PRIVATE_KEY_PASSPHRASE` | No | OCI private key passphrase
`OCI_REGION` | Yes | OCI region
`OCI_TENANCY_OCID` | Yes | OCI tenancy OCID
`OCI_USER_OCID` | Yes | OCI user OCID

For example, to run the script for OCI DNS with the environment name 'test999'...

```
./2-install-rancher-oci-dns.sh -n test999
```

## Install Keycloak

Run the `./4-install-keycloak.sh` script to install Keycloak with the same OCI DNS zone name and environment name from above.  For example ...
```
./4-install-keycloak.sh -n test999 -d oci -s <oci-dns-zone-name>
```

## Install Verrazzano

Run the `./3-install-verrazzano.sh` script to install Verrazzano with the same OCI DNS zone name and environment name from above.  For example ...
```
./3-install-verrazzano.sh -n test999 -d oci -s <oci-dns-zone-name>
```

# Hello World Demo

```text
kubectl apply -f ../examples/hello-helidon/hello-world-model.yaml
kubectl apply -f ../examples/hello-helidon/hello-world-binding.yaml

# Get the LoadBalancer IP address
export LB_IP=$(kubectl get service -n istio-system istio-ingressgateway -o json | jq -r .status.loadBalancer.ingress[0].ip)

# Test endpoint
curl -X GET http://$LB_IP/greet
{“message”:“Hello World!“}
```


# To install in a KIND cluster

You can create a Kind cluster using the sample scripts provided in this project:
```
make create-kind-cluster
```

Or you can use an existing Kind cluster.  Perform the following step to pre-load the docker images into cluster:
```
./misc/kind/load-images.sh
```

Perform the following steps to complete the installation:
```
make install-verrazzano-kind
```
