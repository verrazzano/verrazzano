
# Installation

You can install Verrazzano in a single [Oracle Cloud Infrastructure Container Engine for Kubernetes (OKE)](https://docs.cloud.oracle.com/en-us/iaas/Content/ContEng/Concepts/contengoverview.htm) cluster or
an [Oracle Linux Cloud Native Environment (OCLNE)](https://docs.oracle.com/en/operating-systems/olcne/) deployment. For an Oracle OKE cluster, you have two DNS choices:
[xip.io](http://xip.io/) or
[Oracle OCI DNS](https://docs.cloud.oracle.com/en-us/iaas/Content/DNS/Concepts/dnszonemanagement.htm). Oracle Linux Cloud Native Environment currently supports only a manual DNS.

This README describes installing Verrazzano in an OKE cluster. For instructions on installing Verrazzano on OLCNE, see this [document](install-olcne.md).


> **NOTE**: You should only install this alpha release of Verrazzano in a cluster that can be safely deleted when your evaluation is complete.

## Software requirements

The following software must be installed on your system.  
* curl
* helm (version 3.0.x, 3.1.x or 3.2.x)
* jq
* kubectl
* openssl
* patch (for OCI DNS installation)


### 1. Preparing for installation

* Create the OKE cluster using the OCI console or some other means.  
* Select `v1.16.8` in `KUBERNETES VERSION`.
* An OKE cluster with 3 nodes of `VM.Standard2.4` [OCI Compute instance shape](https://www.oracle.com/cloud/compute/virtual-machines.html) has proven sufficient to install Verrazzano and deploy the Bob's Books example application.

* Set the following `ENV` vars:

```
   export CLUSTER_TYPE=OKE
   export VERRAZZANO_KUBECONFIG=<path to valid kubernetes config>
   export KUBECONFIG=$VERRAZZANO_KUBECONFIG

```

### 2. Do the install

According to your DNS choice, install Verrazzano using one of the following methods:


#### Install using xip.io
Run the following scripts in order:
```
   ./1-install-istio.sh
   ./2a-install-system-components-magicdns.sh
   ./3-install-verrazzano.sh
   ./4-install-keycloak.sh
```


#### Install using OCI DNS


#### Prerequisites
* A DNS Zone is a distinct portion of a domain namespace. Therefore, ensure that the zone is appropriately associated with a parent domain.
For example, an appropriate zone name for parent domain `v8o.example.com` domain is `us.v8o.example.com`.
* Create an OCI DNS zone using the OCI Console or the OCI CLI.  CLI example:
```
oci dns zone create -c <compartment ocid> --name <zone-name-prefix>.v8o.oracledx.com --zone-type PRIMARY
```

#### Installation

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
be used as part of the domain name used to access Verrazzano ingresses.  For example, you could use `sales` as an env-name,
yielding `sales.us.v8o.example.com` as the sales related domain (assuming the domain and zone names listed above).

Run the following scripts in order:
```
   ./1-install-istio.sh
   ./2b-install-system-components-ocidns.sh -n <env-name>
   ./3-install-verrazzano.sh -n <env-name> -d oci -s <oci-dns-zone-name>
   ./4-install-keycloak.sh -n <env-name> -d oci -s <oci-dns-zone-name>
```


### 3. Verify the install

Verrazzano installs multiple objects in multiple namespaces.  All the pods in the `verrazzano-system` namespaces in the `Running` status does not guarantee but likely indicates that Verrazzano is up and running.
```
kubectl get pods -n verrazzano-system
verrazzano-admission-controller-84d6bc647c-7b8tl   1/1     Running   0          5m13s
verrazzano-cluster-operator-57fb95fc99-kqjll       1/1     Running   0          5m13s
verrazzano-monitoring-operator-7cb5947f4c-x9kfc    1/1     Running   0          5m13s
verrazzano-operator-b6d95b4c4-sxprv                1/1     Running   0          5m13s
vmi-system-api-7c8654dc76-2bdll                    1/1     Running   0          4m44s
vmi-system-es-data-0-6679cf99f4-9p25f              2/2     Running   0          4m44s
vmi-system-es-data-1-8588867569-zlwwx              2/2     Running   0          4m44s
vmi-system-es-ingest-78f6dfddfc-2v5nc              1/1     Running   0          4m44s
vmi-system-es-master-0                             1/1     Running   0          4m44s
vmi-system-es-master-1                             1/1     Running   0          4m44s
vmi-system-es-master-2                             1/1     Running   0          4m44s
vmi-system-grafana-5f7bc8b676-xx49f                1/1     Running   0          4m44s
vmi-system-kibana-649466fcf8-4n8ct                 1/1     Running   0          4m44s
vmi-system-prometheus-0-7f97ff97dc-gfclv           3/3     Running   0          4m44s
vmi-system-prometheus-gw-7cb9df774-48g4b           1/1     Running   0          4m44s
```

### 4. Get the console URLs
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

### 5. Get console credentials


You will need the credentials to access the various consoles installed by Verrazzano.

#### Consoles accessed by the same user name/password
- Grafana
- Prometheus
- Kibana
- Elasticsearch

**User:**  `verrazzano`

Run the following command to get the password:

`kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo`


#### The Keycloak admin console

**User:** `keycloakadmin`

Run the following command to get the password:  

`kubectl get secret --namespace keycloak keycloak-http -o jsonpath={.data.password} | base64 --decode; echo`


#### The Rancher console

**User:** `admin`

Run the following command to get the password:  

`kubectl get secret --namespace cattle-system rancher-admin-secret -o jsonpath={.data.password} | base64 --decode; echo`


### 6. (Optional) Install the example applications
Example applications are located in the `examples` directory.


### Known Issues
#### OKE Missing Security List Ingress Rules

The install scripts will perform a check which attempts access through the ingress ports.  If the check fails then the install will exit and you should see error messages like this:

`ERROR: Port 443 is NOT accessible on ingress(132.145.66.80)!  Check that security lists include an ingress rule for the node port 31739.`

On an OKE install this may indicate that there is a missing ingress rule(s).  To check and fix the issue do the following:
  1. Get the ports for the LoadBalancer services.
     * Run `kubectl get services -A`.
     * Note the ports for the LoadBalancer type services.  For example `80:31541/TCP,443:31739/TCP`.
  2. Check the security lists in OCI console.
     * Go to `Networking/Virtual Cloud Networks`.
     * Select the related VCN.
     * Go to the `Security Lists` for the VCN.
     * Select the security list named `oke-wkr-...`.
     * Check the ingress rules for the security list.  There should be one rule for each of the destination ports named in the LoadBalancer services.  In the above example, the destination ports are `31541` & `31739` and we would expect the ingress rule for `31739` to be missing since it was named in the above ERROR output.
     * If a rule is missing, then add it by clicking `Add Ingress Rules` and filling in the source CIDR and destination port range (missing port).  Use the existing rules as a guide.
