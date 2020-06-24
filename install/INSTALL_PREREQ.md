# System Requirements

## Resource Requirements

The following configuration has proven sufficient to install Verrazzano and deploy the Bob's Books example application.

[OCI Compute instance shape](https://www.oracle.com/cloud/compute/virtual-machines.html) `VM.Standard2.4` which has:
* 4 `2.0 GHz Intel® Xeon® Platinum 8167M` cores
* 60 GB of memory
* Select disk size of at least 200 GB.  A minimum of 100 GB of storage required for docker images.

## Software Requirements

The following software must be installed on your system.  
* helm
* jq
* kubectl
* kind (for KinD installation)
* openssl

## Preparing a Cluster

###  Using OKE Cluster

Create the OKE cluster using the OCI console or some other means, then set the following ENV vars:

```
   export CLUSTER_TYPE=OKE
   export VERRAZZANO_KUBECONFIG=<path to valid kubernetes config>
   export KUBECONFIG=$VERRAZZANO_KUBECONFIG

```

### Using kind Cluster

Set the following ENV vars, the run the script to create your kind cluster.

```
   export CLUSTER_TYPE=KIND`
   export VERRAZZANO_KUBECONFIG=<path to kubernetes config where kind cluster info will be written>`
   export KUBECONFIG=$VERRAZZANO_KUBECONFIG`

   ./0-create-kind-cluster.sh`
```

## Create Oracle Container Registry secret

You need to create the "ocr" secret, which is needed for pulling images from the container-registry.oracle.com repository.
```
   kubectl create secret docker-registry ocr \
       --docker-username=<username> \
       --docker-password=<password> \
       --docker-server=container-registry.oracle.com
```

