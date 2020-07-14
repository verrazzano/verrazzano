
# Installation

Verrazzano can be installed in a single [Oracle OKE](https://docs.cloud.oracle.com/en-us/iaas/Content/ContEng/Concepts/contengoverview.htm) cluster 
or a [kind](https://kind.sigs.k8s.io/) cluster. For each cluster type, you have two DNS choices: 
[xip.io](http://xip.io/) or
[Oracle OCI DNS](https://docs.cloud.oracle.com/en-us/iaas/Content/DNS/Concepts/dnszonemanagement.htm).

> **NOTE**: You should only install this alpla release of Verazzano in a cluster that can be safely deleted when your evaluation is complete.

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
* kind (for KinD installation)
* openssl
* patch (for OCI DNS installation)

## 1. Preparing for installation

Prepare for installation as shown below, depending on your cluster type.
Then, create the the docker registry secret.

###  Using an OKE Cluster
Create the OKE cluster using the OCI console or some other means, then set the following ENV vars:
```
   export CLUSTER_TYPE=OKE
   export VERRAZZANO_KUBECONFIG=<path to valid kubernetes config>
   export KUBECONFIG=$VERRAZZANO_KUBECONFIG

```

### Using a kind Cluster
Set the following ENV vars: 
```
   export CLUSTER_TYPE=KIND
   export VERRAZZANO_KUBECONFIG=<path to kubernetes config where kind cluster info will be written>
   export KUBECONFIG=$VERRAZZANO_KUBECONFIG
```

Run the script to create your kind cluster:
```
   ./0-create-kind-cluster.sh
```

### Using an OLCNE Cluster
```
   export CLUSTER_TYPE=OLCNE
   export VERRAZZANO_KUBECONFIG=<path to valid kubernetes config>
   export KUBECONFIG=$VERRAZZANO_KUBECONFIG
```

### Create Oracle Container Registry secret
For all cluster types, you need to create the "ocr" secret. This is needed for pulling images from the container-registry.oracle.com repository.
```
   kubectl create secret docker-registry ocr \
       --docker-username=<username> \
       --docker-password=<password> \
       --docker-server=container-registry.oracle.com
```

## 2. Do the install

Install using xip.io or OCI DNS (2a or 2b).  In both cases, DNS records
will be automatically configured for you.

### 2a. Install using xip.io
Run the following scripts in order:
```
   ./1-install-istio.sh
   ./2a-install-system-components-magicdns.sh
   ./3-install-verrazzano.sh
   ./4-install-keycloak.sh
```

In order to use the consoles, you need to import the root certificate so that the certificates used
by Verrazzano will be trusted by the browser.  Following are examples of importing the certificate on macOS. 

1. Save the root certificate to a file named `ca.crt`:
```kubectl get secret default-secret -n verrazzano-system  -o json|jq '.data."ca.crt"' | tr -d '"' | base64 -D >./ca.crt```

2. Install the root certificate as shown below:
 
    2.1 Firefox: 
Go to `about:preferences#privacy` and click `View Certificates`.  Then click the `Authorities` tab and import the certificate.
On the dialog box that appears, check `Trust this CA to identify websites` and click `OK`.  Then click `OK` on the main dialog box. 

    2.2 Chrome on macOS: 
Browse to `chrome://settings/privacy` and click `Manage certificates`, which opens the keychain.
In the keychain dialog box, select `System` keychain in the top left column, select `Certificates` in the bottom left column, then click the
plus sign on the bottom to import the certificate. Next, double click on the imported certificate and you will see another dialog box.
Expand the trust selection on the left, change `Secure Socket Layers (SSL)` to `Always Trust`.  Finally, dismiss the dialog box to save your change.

**OR**
### 2b. Install using OCI DNS

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
   ./1-install-istio.sh
   ./2b-install-system-components-ocidns.sh -n <env-name>
   ./3-install-verrazzano.sh -n <env-name> -d oci -s <oci-dns-zone-name>
   ./4-install-keycloak.sh -n <env-name> -d oci -s <oci-dns-zone-name>
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
   verrazzano-system   verrazzano-console-ingress         console.myenv.mydomain.com                     128.234.33.198   80, 443   81m
   verrazzano-system   verrazzano-consoleplugin-ingress   verrazzano-consoleplugin.myenv.mydomain.com    128.234.33.198   80, 443   81m
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
- UI Console
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
