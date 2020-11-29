
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

* Install the installation operator.

```
    kubectl apply -f ../deploy/operator.yaml
```

### 2. Do the install

According to your DNS choice, install Verrazzano using one of the following methods.

See [Verrazzano Custom Resource](README.md#7-verrazzano-custom-resource) for a complete description of Verrazzano configuration options.


#### Install using xip.io
The file [install-default.yaml](../config/samples/install-default.yaml) is a template of a Verrazzano custom resource to perform a default xip.io installation.

Run the following commands:
```
    kubectl apply -f ../config/samples/install-default.yaml
    kubectl wait --timeout=20m --for=condition=InstallComplete verrazzano/my-verrazzano
```
Run the following command to monitor the console log output of the installation:
```
    kubectl logs -f $(kubectl get pod -l job-name=verrazzano-install-my-verrazzano -o jsonpath="{.items[0].metadata.name}")
```

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
The file [install-oci.yaml](../config/samples/install-oci.yaml) is a template of a Verrazzano custom resource to perform an OCI DNS installation. Edit this custom resource and provide values for the following configuration settings.

* `spec.environmentName`
* `spec.certificate.acme.emailAddress`
* `spec.dns.oci.ociConfigSecret`
* `spec.dns.oci.dnsZoneCompartmentOCID`
* `spec.dns.oci.dnsZoneOCID`
* `spec.dns.oci.dnsZoneName`

See the table [Verrazzano Custom Resource Definition](README.md#table-verrazzano-custom-resource-definition) for a description of the Verrazzano custom resource inputs.

When you use the OCI DNS installation, you need to provide a Verrazzano name in the configuration
file (`environmentName`) that will be used as part of the domain name used to access Verrazzano
ingresses.  For example, you could use `sales` as an `environmentName`, yielding
`sales.us.v8o.example.com` as the sales-related domain (assuming the domain and zone names listed
previously).

Run the following commands:
```
    kubectl apply -f ../config/samples/install-oci.yaml
    kubectl wait --timeout=20m --for=condition=InstallComplete verrazzano/my-verrazzano
```
Run the following command to monitor the console log output of the installation:
```
    kubectl logs -f $(kubectl get pod -l job-name=verrazzano-install-my-verrazzano -o jsonpath="{.items[0].metadata.name}")
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

### 7. Verrazzano Custom Resource
The Verrazzano custom resource contains the configuration information to perform an installation.

The general format of the yaml to define a Verrazzano custom resource:
```yaml
apiVersion: install.verrazzano.io/v1alpha1
kind: Verrazzano
metadata:
  name: <kubenetes object name>
spec:
  environmentName: <environment name>
  profile: <installation profile>
  dns:
    oci:
      ociConfigSecret: ociConfigSecret
      dnsZoneCompartmentOCID: <DNS compartment OCID>
      dnsZoneOCID: <DNS zone OCID>
      dnsZoneName: <DNS zone name>
    external:
      suffix: <dns suffix>
  ingress:
    type: <load balancer type>
    verrazzano:
      nginxInstallArgs:
        - name: <name of nginx helm parameter>
          value: <value of nging helm parameter>
        - name: <name of nginx helm parameter>
          valueList:
            - <list of values for nginx helm parameter>
      ports:
        - <service port definition>
    application:
      istioInstallArgs:
        - name: <name of Istio helm parameter>
          value: <value of Istio helm parameter>
        - name: <name of Istio helm parameter>
          valueList:
            - <list of values for Istio helm parameter>
  certificate:
    acme:
      provider: <name of certificate issuer>
      emailAddress: <email address>
      environment: <name of environment>
    ca:
      secretName: <name of the secret>
      clusterResourceNamespace: <namespace of the secret>
```

#### Table: Verrazzano Custom Resource Definition
| Configuration setting | Required | Description
| --- | --- | --- |
| spec.environmentName | No | Name of the installation.  This name is part of the endpoint access URL's that are generated. The default value is `default`. |
| spec.profile | No | The installation profile to select.  Valid values are `prod` (production) and `dev` (development).  The default is `prod`. |
| spec.dns.oci | No | This portion of the configuration is specified when using OCI DNS.  This configuration cannot be specified in conjunction with spec.dns.external.  |
| spec.dns.oci.ociConfigSecret | Yes | Name of the OCI configuration secret.  Generate a secret named "oci-config" based on the OCI configuration profile you wish to leverage.  You can specify a profile other than DEFAULT and a different secret name if you wish.  See instructions by executing ./install/create_oci_config_secret.sh.|
| spec.dns.oci.dnsZoneCompartmentOCID | Yes | The OCI DNS compartment OCID. |
| spec.dns.oci.dnsZoneOCID | Yes | The OCI DNS zone OCID. |
| spec.dns.oci.dnsZoneName | Yes | Name of OCI DNS zone. |
| spec.dns.external | No | This portion of the configuration is specified when using OLCNE.  This configuration cannot be specified in conjunction with spec.dns.oci. |
| spec.dns.external.suffix | Yes | The suffix for DNS names. |
| spec.ingress | No | This portion of the configuration defines the ingress. |
| spec.ingress.type | No | The ingress type.  Valid values are `LoadBalancer` and `NodePort`.  The default value is `LoadBalancer`. |
| spec.ingress.verrazzano | No | This portion of the configuration defines the ingress for the Verrazzano infrastructure endpoints. |
| spec.ingress.verrazzano.nginxInstallArgs | No | A list of Nginx helm chart arguments and values to apply during the installation of Nginx.  Each argument is specified as either a `name/value` or `name/valueList` pair. |
| spec.ingress.verrazzano.ports | No | The list of ports for the ingress. Each port definition is of type [ServicePort](https://godoc.org/k8s.io/api/core/v1#ServicePort). |
| spec.ingress.application | No | This portion of the configuration defines the ingress for the application endpoints. |
| spec.ingress.application.istioInstallArgs | No | A list of Istio helm chart arguments and values to apply during the installation of Istio.  Each argument is specified as either a `name/value` or `name/valueList` pair. |
| spec.certificate | No | This portion of the configuration defines the certificate information. |
| spec.certificate.acme | No | Define a certificate issued by `acme`. |
| spec.certificate.acme.provider | Yes | The certificate issuer provider. |
| spec.certificate.acme.emailAddress | No | Email address. |
| spec.certificate.acme.environment | No | The name of the environment. |
| spec.certificate.ca | No | Define a certificate issued by `ca`. |
| spec.certificate.ca.secretName | Yes | Name of the secret. |
| spec.certificate.ca.clusterResourceNamespace | Yes | The namespace of the secret. |


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
