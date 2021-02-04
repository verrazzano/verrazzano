# Verrazzano Installation on Oracle Linux Cloud Native Environment

This document describes installing [Verrazzano](https://verrazzano.io/) on [Oracle Linux Cloud Native Environment (OLCNE)](https://docs.oracle.com/en/operating-systems/olcne/)
cluster.

> **NOTE**: You should only install this alpha release of Verrazzano in a cluster that can be safely deleted when your evaluation is complete.

## Prerequisites
* Verrazzano requires Oracle Linux Cloud Native Environment version 1.1 or later.
* Verrazzano installation requires that the following software be installed on the system used to install Verrazzano.
  The Oracle Linux Cloud Native Environment operator node is typically used to install Verrazzano.  
  * curl
  * helm (version 3.0.x, 3.1.x or 3.2.x)
  * jq
  * kubectl
  * openssl

  If you are using Oracle Linux 7, then these requirements can be installed as shown:
  ```
  sudo yum install -y oracle-olcne-release-el7
  sudo yum-config-manager --enable ol7_olcne11 ol7_addons ol7_latest
  sudo yum install -y kubectl helm jq openssl curl
  ```

## Installation

### 1. Prepare for the Oracle Linux Cloud Native Environment install
Oracle Linux Cloud Native Environment can be installed in several different types of environments.
These range from physical, on-premises hardware to virtualized cloud infrastructure.
The Oracle Linux Cloud Native Environment installation instructions assume that networking and compute resources already exist.
The basic infrastructure requirements are a network with a public and private subnet
and a set of hosts connected to those networks.

#### OCI Example
The following is an example of OCI infrastructure that can be used to evaluate Verrazzano installed on Oracle Linux Cloud Native Environment.
If other environments are used, the capacity and configuration should be similar.

You can use the VCN Wizard of the OCI Console to automatically create most of the described network infrastructure.
Additional security lists/rules, as detailed below, need to be added manually.
All CIDR values provided are examples and can be customized as required.

* Virtual Cloud Network (for example, CIDR 10.0.0.0/16)
  * Public Subnet (for example, CIDR 10.0.0.0/24)
    * Security List / Ingress Rules

      |Stateless|Destination|Protocol|Source Ports|Destination Ports|Type & Code|Description        |
      |---------|-----------|--------|------------|-----------------|-----------|-------------------|
      |No       |`0.0.0.0/0`  |ICMP    |            |                 |3, 4       |ICMP errors        |
      |No       |`10.0.0.0/16`|ICMP    |            |                 |3          |ICMP errors        |
      |No       |`0.0.0.0/0`  |TCP     |All         |22               |           |SSH                |
      |No       |`0.0.0.0/0`  |TCP     |All         |80               |           |HTTP load balancer |
      |No       |`0.0.0.0/0`  |TCP     |All         |443              |           |HTTPS load balancer|

    * Security List / Egress Rules

      |Stateless|Destination|Protocol|Source Ports|Destination Ports|Type & Code|Description        |
      |---------|-----------|--------|------------|-----------------|-----------|-------------------|
      |No       |`10.0.1.0/24`|TCP     |All         |22               |           |SSH                |
      |No       |`10.0.1.0/24`|TCP     |All         |30080            |           |HTTP load balancer |
      |No       |`10.0.1.0/24`|TCP     |All         |30443            |           |HTTPS load balancer|
      |No       |`10.0.1.0/24`|TCP     |All         |31380            |           |HTTP load balancer |
      |No       |`10.0.1.0/24`|TCP     |All         |31390            |           |HTTPS load balancer|

  * Private Subnet (for example, CIDR 10.0.1.0/24)
    * Security List / Ingress Rules

      |Stateless|Destination|Protocol|Source Ports|Destination Ports|Type & Code|Description          |
      |---------|-----------|--------|------------|-----------------|-----------|---------------------|
      |No       |`0.0.0.0/0`  |ICMP    |            |                 |3, 4       |ICMP errors          |
      |No       |`10.0.0.0/16`|ICMP    |            |                 |3          |ICMP errors          |
      |No       |`10.0.0.0/16`|TCP     |All         |22               |           |SSH                  |
      |No       |`10.0.0.0/24`|TCP     |All         |30080            |           |HTTP load balancer   |
      |No       |`10.0.0.0/24`|TCP     |All         |30443            |           |HTTPS load balancer  |
      |No       |`10.0.0.0/24`|TCP     |All         |31380            |           |HTTP load balancer   |
      |No       |`10.0.0.0/24`|TCP     |All         |31390            |           |HTTPS load balancer  |
      |No       |`10.0.1.0/24`|UDP     |All         |111              |           |NFS                  |
      |No       |`10.0.1.0/24`|TCP     |All         |111              |           |NFS                  |
      |No       |`10.0.1.0/24`|UDP     |All         |2048             |           |NFS                  |
      |No       |`10.0.1.0/24`|TCP     |All         |2048-2050        |           |NFS                  |
      |No       |`10.0.1.0/24`|TCP     |All         |2379-2380        |           |Kubernetes etcd      |
      |No       |`10.0.1.0/24`|TCP     |All         |6443             |           |Kubernetes API Server|
      |No       |`10.0.1.0/24`|TCP     |All         |6446             |           |MySQL                |
      |No       |`10.0.1.0/24`|TCP     |All         |8090-8091        |           |OLCNE Platform Agent |
      |No       |`10.0.1.0/24`|UDP     |All         |8472             |           |Flannel              |
      |No       |`10.0.1.0/24`|TCP     |All         |10250-10255      |           |Kubernetes Kublet    |

    * Security List / Egress Rules

      |Stateless|Destination|Protocol|Source Ports|Destination Ports|Type and Code|Description       |
      |---------|-----------|--------|------------|-----------------|-------------|------------------|
      |No       |`10.0.0.0/0` |TCP     |            |                 |             |All egress traffic|

  * DHCP Options

    |DNS Type                 |
    |-------------------------|
    |Internet and VCN Resolver|

  * Route Tables
    * Public Subnet Route Table Rules

      |Destination|Target          |
      |-----------|----------------|
      |`0.0.0.0/0`  |Internet Gateway|

    * Private Subnet Route Table Rules

      |Destination     |Target         |
      |----------------|---------------|
      |`0.0.0.0/0`       |NAT Gateway    |
      |All OCI Services|Service Gateway|

  * Internet Gateway
  * NAT Gateway
  * Service Gateway

The following compute resources adhere to the guidelines provided in the Oracle Linux Cloud Native Environment [Getting Started](https://docs.oracle.com/en/operating-systems/olcne/start/deploy-kube.html) guide.
The attributes indicated (for example, Subnet, RAM, Shape, and Image) are recommendations that have been tested.
Other values can be used if required.

* Compute Instances

  | Role                          | Subnet  | Suggested RAM | Compatible VM Shape | Compatible VM Image |
  |-------------------------------|---------|---------------|---------------------|---------------------|
  | SSH Jump Host                 | Public  | 8GB           | VM.Standard.E2.1    | Oracle Linux 7.8    |
  | OLCNE Operator Host           | Private | 16GB          | VM.Standard.E2.2    | Oracle Linux 7.8    |
  | Kubernetes Control Plane Node | Private | 32GB          | VM.Standard.E2.4    | Oracle Linux 7.8    |
  | Kubernetes Worker Node 1      | Private | 32GB          | VM.Standard.E2.4    | Oracle Linux 7.8    |
  | Kubernetes Worker Node 2      | Private | 32GB          | VM.Standard.E2.4    | Oracle Linux 7.8    |
  | Kubernetes Worker Node 3      | Private | 32GB          | VM.Standard.E2.4    | Oracle Linux 7.8    |

### 2. Do the Oracle Linux Cloud Native Environment install
Deploy Oracle Linux Cloud Native Environment with the Kubernetes module, following instructions from the [Getting Started](https://docs.oracle.com/en/operating-systems/olcne/start/deploy-kube.html) guide.
* Use a single Kubernetes control plane node
* Skip the Kubernetes API load balancer ([3.4.3](https://docs.oracle.com/en/operating-systems/olcne/start/install-lb.html))
* Use private CA certificates ([3.5.3](https://docs.oracle.com/en/operating-systems/olcne/start/certs-private.html))

### 3. Prepare for the Verrazzano install

####  Prerequisites Overview
A Verrazzano Oracle Linux Cloud Native Environment deployment requires:
* A default storage provider that supports "Multiple Read/Write" mounts. For example, an NFS service like:
    * Oracle Cloud Infrastructure File Storage Service.
    * A hardware-based storage system that provides NFS capabilities.
* Load balancers in front of the worker nodes in the cluster.
* DNS records that reference the load balancers.

Examples for meeting these requirements follow.

#### Prerequisites Detail

##### Storage
Verrazzano requires persistent storage for several components.
This persistent storage is provided by a default storage class.
A number of persistent storage providers exist for Kubernetes.
This guide will focus on pre-allocated persistent volumes.
In particular, the provided samples will illustrate the use of OCI's NFS File System.

###### OCI Example  
Before storage can be exposed to Kubernetes, it must be created.
In OCI, this is done using File System resources.
Using the OCI Console, create a new File System.
Within the new File System, create an Export.
Remember the value used for  `Export Path` as it will be used later.
Also note the Mount Target's `IP Address` for use later.

After the exports have been created, referenced persistent volume folders (for example, `/example/pv0001`) will need to be created.
In OCI, this can be done by mounting the export on one of the Kubernetes worker nodes and creating the folders.
In the following example, the value `/example` is the `Export Path` and `10.0.1.8` is the Mount Target's `IP Address`.
The following command should be run on one of the Kubernetes worker nodes.
This will result in the creation of nine persistent volume folders.
The reason for nine persistent volume folders is covered in the next section.
```
sudo mount 10.0.1.8:/example /mnt
for x in {0001..0009}; do sudo mkdir -p /mnt/pv${x} && sudo chmod 777 /mnt/pv${x}; done
```

###### Persistent Volumes
A default Kubernetes storage class is required by Verrazzano.
When using pre-allocated `PersistentVolumes`, for example NFS, persistent volumes should be declared as following.
The value for `name` may be customized but will need to match the `PersistentVolume` `storageClassName` value later.
* Create a default `StorageClass`
  ```
  cat << EOF | kubectl apply -f -
    apiVersion: storage.k8s.io/v1
    kind: StorageClass
    metadata:
      name: example-nfs
      annotations:
        storageclass.kubernetes.io/is-default-class: "true"
    provisioner: kubernetes.io/no-provisioner
    volumeBindingMode: WaitForFirstConsumer
  EOF
  ```
* Create the required number of `PersistentVolume` resources.
  The Verrazzano system requires five persistent volumes for itself.
  Each deployed Verrazzano binding requires an additional four persistent volumes.
  The following command creates nine persistent volumes, which is enough for one deployed binding.
  The value for `storageClassName` must match the above `StorageClass` name.
  The values for `name` may be customized.
  The value for `path` must match the `Export Path` of the Export from above, combined with the persistent volume folder from above.
  The value for `server` must be changed to match the location of your file system server.  
  ```
  for n in {0001..0009}; do cat << EOF | kubectl apply -f -
    apiVersion: v1
    kind: PersistentVolume
    metadata:
      name: pv${n}
    spec:
      storageClassName: example-nfs
      accessModes:
        - ReadWriteOnce
        - ReadWriteMany
      capacity:
        storage: 50Gi
      nfs:
        path: /example/pv${n}
        server: 10.0.1.8
      volumeMode: Filesystem
      persistentVolumeReclaimPolicy: Recycle
  EOF
  ```

#### Load Balancers
Verrazzano on Oracle Linux Cloud Native Environment uses external load balancer services.
These will not automatically be provided by Verrazzano or Kubernetes.
Two load balancers must be deployed outside of the subnet used for the Kubernetes cluster.
One load balancer is for management traffic and the other for application traffic.

Specific steps will differ for each load balancer provider, but a generic configuration and an OCI example follow.

##### Generic Configuration
* Target Host: Host names of Kubernetes worker nodes
* Target Ports: See table
* External Ports: See table
* Distribution: Round Robin
* Health Check: TCP

| Traffic Type | Service Name                                  | Type | Suggested External Port | Target Port |
|--------------|-----------------------------------------------|------|-------------------------|-------------|
| Application  | `istio-ingressgateway`                        | TCP  | 80                      | 31380       |
| Application  | `istio-ingressgateway`                        | TCP  | 443                     | 31390       |
| Management   | `ingress-controller-nginx-ingress-controller` | TCP  | 80                      | 30080       |
| Management   | `ingress-controller-nginx-ingress-controller` | TCP  | 443                     | 30443       |

##### OCI Example
The following details can be used to create OCI load balancers for accessing application and management user interfaces, respectively.
These load balancers will route HTTP/HTTPS traffic from the Internet to the private subnet.
If load balancers are desired, then they should be created now even though the application and management endpoints will be installed later.

* Application Load Balancer: Public Subnet
  * Listeners
    * HTTP Listener: Protocol TCP, Port 80
    * HTTPS Listener: Protocol TCP, Port 443
  * Backend Sets
    * HTTP Backend Sets:
      * Health Check: Protocol TCP, Port 31380
      * Backends: Kubernetes Worker Nodes, Port 31380, Distribution Policy Weighted Round Robin
    * HTTPS Backend Sets
      * Health Check: Protocol TCP, Port 31390
      * Backends: Kubernetes Worker Nodes, Port 31390, Distribution Policy Weighted Round Robin
* Management Load Balancer: Public Subnet
  * Listeners
    * HTTP Listener: Protocol TCP, Port 80
    * HTTPS Listener: Protocol TCP, Port 443
  * Backend Sets
    * HTTP Backend Sets:
      * Health Check: Protocol TCP, Port 30080
      * Backends: Kubernetes Worker Nodes, Port 30080, Distribution Policy Weighted Round Robin
    * HTTPS Backend Sets
      * Health Check: Protocol TCP, Port 30443
      * Backends: Kubernetes Worker Nodes, Port 30443, Distribution Policy Weighted Round Robin

##### DNS
When using the `spec.dns.external` DNS type, the installer searches the DNS zone you provide for two specific A records.
These are used to configure the cluster and should refer to external addresses of the load balancers in the previous step.
The A records will need to be created manually.

**NOTE:** At this time, the only supported deployment for Oracle Linux Cloud Native Environment is the external DNS type.

|Record             | Use                                                                                              |
|-------------------|--------------------------------------------------------------------------------------------------|
|ingress-mgmt       | Set as the `.spec.externalIPs` value of the `ingress-controller-nginx-ingress-controller` service|
|ingress-verrazzano | Set as the `.spec.externalIPs` value of the `istio-ingressgateway` service                       |

For example:
```
198.51.100.10                                   A       ingress-mgmt.myenv.mydomain.com.
203.0.113.10                                    A       ingress-verrazzano.myenv.mydomain.com.
```

Verrazzano installation will result in a number of management services that need to point to the ingress-mgmt address.
```
keycloak.myenv.mydomain.com                      CNAME   ingress-mgmt.myenv.mydomain.com.
rancher.myenv.mydomain.com                       CNAME   ingress-mgmt.myenv.mydomain.com.

grafana.vmi.system.myenv.mydomain.com            CNAME   ingress-mgmt.myenv.mydomain.com.
prometheus.vmi.system.myenv.mydomain.com         CNAME   ingress-mgmt.myenv.mydomain.com.
kibana.vmi.system.myenv.mydomain.com             CNAME   ingress-mgmt.myenv.mydomain.com.
elasticsearch.vmi.system.myenv.mydomain.com      CNAME   ingress-mgmt.myenv.mydomain.com.
```

Deployment of applications as a VerrazzanoBinding will create four more services in the form:
* grafana.vmi.**mybinding**.myenv.mydomain.com
* prometheus.vmi.**mybinding**.myenv.mydomain.com
* kibana.vmi.**mybinding**.myenv.mydomain.com
* elasticsearch.vmi.**mybinding**.myenv.mydomain.com

For simplicity, an administrator may want to create [wildcard DNS records](https://tools.ietf.org/html/rfc1034#section-4.3.3) for the management addresses:
```
*.system.myenv.mydomain.com                      CNAME   ingress-mgmt.myenv.mydomain.com.
*.mybinding.myenv.mydomain.com                   CNAME   ingress-mgmt.myenv.mydomain.com.
```
or
```
*.myenv.mydomain.com                             CNAME   ingress-mgmt.myenv.mydomain.com.
```

##### OCI Example
DNS is configured in OCI by creating DNS zones in the OCI Console.
When creating a DNS zone, use these values.
* Method: Manual
* Zone Name: `<dns-suffix>`
* Zone Type: Primary

The value for `<dns-suffix>` excludes the environment (for example, use the `mydomain.com` portion of `myenv.mydomain.com`).

DNS A records must be manually added to the zone and published using values described above.
DNS CNAME records, in the same way.

### 4. Do the Verrazzano install
During the Verrazzano install, these steps should be performed on the Oracle Linux Cloud Native Environment operator node.

Clone the Verrazzano install repository.
```
git clone https://github.com/verrazzano/verrazzano.git
cd verrazzano/install
```
If required, use the following commands to install `git`.
```
sudo yum install -y git
```
Edit the sample Verrazzano custom resource [install-olcne.yaml](operator/config/samples/install-olcne.yaml) file and provide the configuration settings for your OLCNE environment as follows:

- The value for `spec.environmentName` is a unique DNS subdomain for the cluster (for example, `myenv` in `myenv.mydomain.com`).
- The value for `spec.dns.external.suffix` is the remainder of the DNS domain (for example, `mydomain.com` in `myenv.mydomain.com`).
- Under `spec.ingress.verrazzano.nginxInstallArgs`, the value for `controller.service.externalIPs` is the IP address of `ingress-mgmt.<myenv>.<mydomain.com>` configured during DNS set up.
- Under  `spec.ingress.application.istioInstallArgs`, the value for `gateways.istio-ingressgateway.externalIPs` is the IP address of `ingress-verrazzano.<myenv>.<mydomain.com>` configured during DNS set up.

You will install Verrazzano using the `external` DNS type (the example custom resource for OLCNE is already configured to use `spec.dns.external`).

Set the following environment variables:

The value for `<path to valid Kubernetes config>` is typically `${HOME}/.kube/config`
```
export KUBECONFIG=$VERRAZZANO_KUBECONFIG
```

Run the following commands:
```
kubectl apply -f operator/deploy/operator.yaml
kubectl apply -f operator/config/samples/install-olcne.yaml
kubectl wait --timeout=20m --for=condition=InstallComplete verrazzano/my-verrazzano
```

Run the following command to monitor the console log output of the installation:
```
    kubectl logs -f $(kubectl get pod -l job-name=verrazzano-install-my-verrazzano -o jsonpath="{.items[0].metadata.name}")
```

### 5. Verify the Verrazzano install
Verrazzano installs multiple objects in multiple namespaces.  All the pods in the `verrazzano-system` namespaces in the `Running` state does not guarantee, but likely indicates that Verrazzano is up and running.
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

### 6. Get the console URLs
Verrazzano installs several consoles.
You can get the ingress for the consoles with the following command:  

`kubectl get ingress -A`

Prefix `https://` to the host name returned to get the URL.
For example, `https://rancher.myenv.mydomain.com`

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

### 7. Get console credentials
You will need the credentials to access the various consoles installed by Verrazzano.

#### Consoles accessed by the same user name/password
* Grafana
* Prometheus
* Kibana
* Elasticsearch

**User:** `verrazzano`

Run the following command to get the password:
```
kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo
```

#### The Keycloak admin console
**User:** `keycloakadmin`

Run the following command to get the password:
```
kubectl get secret --namespace keycloak keycloak-http -o jsonpath={.data.password} | base64 --decode; echo
```

#### The Rancher console
**User:** `admin`

Run the following command to get the password:
```
kubectl get secret --namespace cattle-system rancher-admin-secret -o jsonpath={.data.password} | base64 --decode; echo
```

### 8. (Optional) Install the example applications

Example applications are located in the `examples` directory.
