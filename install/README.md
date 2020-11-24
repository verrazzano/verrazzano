
# Installation

You can install Verrazzano in a single [Oracle Cloud Infrastructure Container Engine for Kubernetes (OKE)](https://docs.cloud.oracle.com/en-us/iaas/Content/ContEng/Concepts/contengoverview.htm) cluster or
an [Oracle Linux Cloud Native Environment (OCLNE)](https://docs.oracle.com/en/operating-systems/olcne/) deployment. For an Oracle OKE cluster, you have two DNS choices:
[xip.io](http://xip.io/) or
[Oracle OCI DNS](https://docs.cloud.oracle.com/en-us/iaas/Content/DNS/Concepts/dnszonemanagement.htm). Oracle Linux Cloud Native Environment currently supports only a manual DNS.

This README describes installing Verrazzano in an OKE cluster. For instructions on installing Verrazzano on OLCNE, see this [document](install-olcne.md).


> **NOTE**: You should install this alpha release of Verrazzano only in a cluster that can be safely deleted when your evaluation is complete.

## Software requirements

The following software must be installed on your system.  
* kubectl


### 1. Prepare for installation

* Create the OKE cluster using the OCI Console or some other means.  
* For `KUBERNETES VERSION`, select `v1.16.8`.
* For `SHAPE`, an OKE cluster with 3 nodes of `VM.Standard2.4` [OCI Compute instance shape](https://www.oracle.com/cloud/compute/virtual-machines.html) has proven sufficient to install Verrazzano and deploy the Bob's Books example application.

* Set the following `ENV` vars:

```
   export KUBECONFIG=<path to valid Kubernetes config>
```

* Create the optional `imagePullSecret` named `verrazzano-container-registry`.  This step is required when one or more of the Docker images installed by Verrazzano are private.  For example, while testing a change to the `verrazzano-operator`, you may be using a Docker image that requires credentials to access it.

```
    kubectl create secret docker-registry verrazzano-container-registry --docker-username=<username> --docker-password=<password> --docker-server=<docker server>
```

### 2. Do the install

According to your DNS choice, install Verrazzano using one of the following methods.


#### Install using xip.io
Run the following scripts in order:
```
   ./1-install-istio.sh
   ./2-install-system-components.sh
   ./3-install-verrazzano.sh
   ./4-install-keycloak.sh
```
This is the default configuration, and will automatically use the configuration file `config/config_defaults.json`

#### Install using OCI DNS


#### Prerequisites
* A DNS zone is a distinct portion of a domain namespace. Therefore, ensure that the zone is appropriately associated with a parent domain.
For example, an appropriate zone name for parent domain `v8o.example.com` domain is `us.v8o.example.com`.
* Create an OCI DNS zone using the OCI Console or the OCI CLI.  

  CLI example:
  ```
  oci dns zone create -c <compartment ocid> --name <zone-name-prefix>.v8o.oracledx.com --zone-type PRIMARY
  ```

#### Installation

Installing Verrazzano on OCI DNS requires some configuration settings to create DNS records.
The configuration file `config/config_oci.json` has a template of the required configuration
information. Edit this file and provide values for the following configuration settings.

Configuration setting | Required | Description
--- | --- | --- |
`certificates.acme.emailAddress` | Yes | Email address
`dns.oci.dnsZoneCompartmentOcid` | Yes | OCI DNS compartment OCID
`dns.oci.dnsZoneName` | Yes | Name of OCI DNS zone
`dns.oci.dnsZoneOcid` | Yes | OCI DNS zone OCID
`dns.oci.fingerprint` | Yes | OCI fingerprint
`dns.oci.privateKeyFile` | Yes | OCI private key file
`dns.oci.privateKeyPassphrase` | No | OCI private key passphrase
`dns.oci.region` | Yes | OCI region
`dns.oci.tenancyOcid` | Yes | OCI tenancy OCID
`dns.oci.userOcid` | Yes | OCI user OCID

When you use the OCI DNS installation, you need to provide a Verrazzano name in the configuration
file (`environmentName`) that will be used as part of the domain name used to access Verrazzano
ingresses.  For example, you could use `sales` as an `environmentName`, yielding
`sales.us.v8o.example.com` as the sales-related domain (assuming the domain and zone names listed
previously).

Set the `INSTALL_CONFIG_FILE` environment variable to the edited OCI configuration file (e.g.)
```
export INSTALL_CONFIG_FILE=./config/config_oci.json
```

Run the following scripts in order:
```
   ./1-install-istio.sh
   ./2-install-system-components.sh
   ./3-install-verrazzano.sh
   ./4-install-keycloak.sh
```


### 3. Verify the install

Verrazzano installs multiple objects in multiple namespaces. In the `verrazzano-system` namespaces, all the pods in the `Running` state does not guarantee, but likely indicates that Verrazzano is up and running.
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

The install scripts perform a check, which attempts access through the ingress ports.  If the check fails, then the install will exit and you will see error messages like this:

`ERROR: Port 443 is NOT accessible on ingress(132.145.66.80)!  Check that security lists include an ingress rule for the node port 31739.`

On an OKE install, this may indicate that there is a missing ingress rule or rules.  To verify and fix the issue, do the following:
  1. Get the ports for the LoadBalancer services.
     * Run `kubectl get services -A`.
     * Note the ports for the LoadBalancer type services.  For example `80:31541/TCP,443:31739/TCP`.
  2. Check the security lists in the OCI Console.
     * Go to `Networking/Virtual Cloud Networks`.
     * Select the related VCN.
     * Go to the `Security Lists` for the VCN.
     * Select the security list named `oke-wkr-...`.
     * Check the ingress rules for the security list.  There should be one rule for each of the destination ports named in the LoadBalancer services.  In the above example, the destination ports are `31541` & `31739`. We would expect the ingress rule for `31739` to be missing because it was named in the ERROR output.
     * If a rule is missing, then add it by clicking `Add Ingress Rules` and filling in the source CIDR and destination port range (missing port).  Use the existing rules as a guide.
