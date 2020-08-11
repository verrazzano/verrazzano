
# Installation

Verrazzano can be installed in a single [Oracle Cloud Infrastructure Container Engine for Kubernetes (OKE)](https://docs.cloud.oracle.com/en-us/iaas/Content/ContEng/Concepts/contengoverview.htm) cluster or
an [Oracle Linux Cloud Native Environment](https://docs.oracle.com/en/operating-systems/olcne/) deployment.
For the Oracle OKE cluster type you have two DNS choices:
[xip.io](http://xip.io/) or
[Oracle OCI DNS](https://docs.cloud.oracle.com/en-us/iaas/Content/DNS/Concepts/dnszonemanagement.htm).
Oracle Linux Cloud Native Environment currently only supports a third choice of manual DNS
as described in the [Verrazzano Prerequisites](https://verrazzano.io/docs/install/prereqs/).

> **NOTE**: You should only install this alpha release of Verrazzano in a cluster that can be safely deleted when your evaluation is complete.

## Resource Requirements

The following configuration has proven sufficient to install Verrazzano and deploy the Bob's Books example application.

[OCI Compute instance shape](https://www.oracle.com/cloud/compute/virtual-machines.html) `VM.Standard2.4` which has:
* 4 `2.0 GHz Intel® Xeon® Platinum 8167M` cores
* 60 GB of memory
* Select disk size of at least 200 GB.  A minimum of 100 GB of storage required for docker images.

## Software Requirements

The following software must be installed on your system.  
* curl
* helm
* jq
* kubectl
* openssl
* patch (for OCI DNS installation)

## 1. Preparing for installation

Prepare for installation as shown below, depending on your cluster type.
Then, create the docker registry secret.

###  Using an OKE Cluster
Create the OKE cluster using the OCI console or some other means, then set the following ENV vars:
```
   export CLUSTER_TYPE=OKE
   export VERRAZZANO_KUBECONFIG=<path to valid kubernetes config>
   export KUBECONFIG=$VERRAZZANO_KUBECONFIG

```

### Using an OLCNE Cluster
```
   export CLUSTER_TYPE=OLCNE
   export VERRAZZANO_KUBECONFIG=<path to valid kubernetes config>
   export KUBECONFIG=$VERRAZZANO_KUBECONFIG
```
> **NOTE**: At this time the only supported deployment for OLCNE is DNS type manual

### Create Oracle Container Registry secret
For all cluster types, you need to create the "ocr" secret. This is needed for pulling images from the container-registry.oracle.com repository.
```
   kubectl create secret docker-registry ocr \
       --docker-username=<username> \
       --docker-password=<password> \
       --docker-server=container-registry.oracle.com
```

## 2. Do the install

Install using xip.io, OCI DNS or manual DNS configuration. In xip.io and OCI DNS cases the DNS records
will be automatically configured for you. In manual mode you must create the records as per
[Verrazzano Prerequisites](https://verrazzano.io/docs/install/prereqs/)

### Install using xip.io
Run the following scripts in order:
```
   ./install/1-install-istio.sh
   ./install/2a-install-system-components-magicdns.sh
   ./install/3-install-verrazzano.sh
   ./install/4-install-keycloak.sh
```
**OR**
### Install using manual DNS
Run the following scripts in order:
```
   ./install/1-install-istio.sh                       -d manual -n <env-name> -s <dns-suffix>
   ./install/2a-install-system-components-magicdns.sh -d manual -n <env-name> -s <dns-suffix>
   ./install/3-install-verrazzano.sh                  -d manual -n <env-name> -s <dns-suffix>
   ./install/4-install-keycloak.sh                    -d manual -n <env-name> -s <dns-suffix>
```
**OR**
### Install using OCI DNS

Installing Verrazzano on OCI DNS requires the following environment variables to create DNS records:

Environment Variable | Required | Description
--- | --- | --- |
`EMAIL_ADDRESS` | Yes | Email address
`OCI_COMPARTMENT_OCID` | Yes | OCI DNS Compartment OCID
`OCI_DNS_ZONE_NAME` | Yes | Name of OCI DNS Zone
`OCI_DNS_ZONE_OCID` | Yes | OCI DNS Zone OCID
`OCI_FINGERPRINT` | Yes | OCI fingerprint
`OCI_PRIVATE_KEY_FILE` | Yes | OCI private key file
`OCI_PRIVATE_KEY_PASSPHRASE` | No | OCI private key passphrase
`OCI_REGION` | Yes | OCI region
`OCI_TENANCY_OCID` | Yes | OCI tenancy OCID
`OCI_USER_OCID` | Yes | OCI user OCID

When you use OCI DNS install, you need to provide a Verrazzano name (env-name) that will
be used as part of the domain name used to access Verrazzano ingresses.  For example, you could use `sales` as an env-name.

Run the following scripts in order:
```
   ./install/1-install-istio.sh
   ./install/2b-install-system-components-ocidns.sh -n <env-name>
   ./install/3-install-verrazzano.sh -n <env-name> -d oci -s <oci-dns-zone-name>
   ./install/4-install-keycloak.sh -n <env-name> -d oci -s <oci-dns-zone-name>
```

## 3. Get the console URLs
Verrazzano installs several consoles.  You can get the ingress for the consoles with the following command:  
`kubectl get ingress -A`

Simply prefix `https://` to the host name to get the URL.  For example `https://rancher.myenv.mydomain.com`

Following is an example of the ingresses:
```
   NAMESPACE           NAME                               HOSTS                                          ADDRESS          PORTS     AGE
   cattle-system       rancher                            rancher.myenv.mydomain.com                     128.234.33.198   80, 443   93m
   keycloak            keycloak                           keycloak.myenv.mydomain.com                    128.234.33.198   80, 443   69m
   verrazzano-system   verrazzano-operator-ingress        api.myenv.mydomain.com                         128.234.33.198   80, 443   81m
   verrazzano-system   vmi-system-api                     api.vmi.system.myenv.mydomain.com              128.234.33.198   80, 443   80m
   verrazzano-system   vmi-system-es-ingest               elasticsearch.vmi.system.myenv.mydomain.com    128.234.33.198   80, 443   80m
   verrazzano-system   vmi-system-grafana                 grafana.vmi.system.myenv.mydomain.com          128.234.33.198   80, 443   80m
   verrazzano-system   vmi-system-kibana                  kibana.vmi.system.myenv.mydomain.com           128.234.33.198   80, 443   80m
   verrazzano-system   vmi-system-prometheus              prometheus.vmi.system.myenv.mydomain.com       128.234.33.198   80, 443   80m
   verrazzano-system   vmi-system-prometheus-gw           prometheus-gw.vmi.system.myenv.mydomain.com    128.234.33.198   80, 443   80m
```

## 4. Get Console Credentials
You will need the credentials to access the various consoles installed by Verrazzano.

### Consoles accessed by the same username/password
- Grafana
- Prometheus
- Kibana
- Elasticsearch

User:  `verrazzano`

Run the following command to get the password: 
`kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo`

### The Keycloak Admin console
User `keycloakadmin`
 
Run the following command to get the password:  
`kubectl get secret --namespace keycloak keycloak-http -o jsonpath={.data.password} | base64 --decode; echo`

### The Rancher console
User `admin`
 
Run the following command to get the password:  
`kubectl get secret --namespace cattle-system rancher-admin-secret -o jsonpath={.data.password} | base64 --decode; echo`


## Install example applications
Example applications can be found in the `examples` folder.


## More Information
For additional information, see the [Verrazzano documentation](https://verrazzano.io/docs).
