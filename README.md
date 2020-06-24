# Verrazzano Enterprise Container Platform
> **NOTE**: This is an early alpha release of Verrazzano. It is suitable for investigation and education usage. It is not suitable for production use. 

## Introduction
Verrazzano Enterprise Container Platform is a curated collection of open source and Oracle-authored components that form a complete platform for modernizing existing applications, and for deploying and managing your container applications across multiple Kubernetes clusters. 

Verrazzano Enterprise Container Platform includes the following capabilities:

- Hybrid and multi-cluster workload management
- Special handling for WebLogic, Coherence, and Helidon applications
- Multi-cluster infrastructure management
- Integrated and pre-wired application monitoring
- Integrated security
- DevOps and GitOps enablement

This repository contains installation scripts and example applications for use with Verrazzano.

> **NOTE**: This is an early alpha release of Verrazzano. Some features are still in development. 

# Installation

Verrazzano can be installed in a single Oracle OKE or [kind](https://kind.sigs.k8s.io/) cluster. For each cluster type, 
you have two DNS choices: [xip.io](http://xip.io/) or
 [Oracle OCI DNS](https://docs.cloud.oracle.com/en-us/iaas/Content/DNS/Concepts/dnszonemanagement.htm).

> **NOTE**: You should only install this alpla release of Verazzano in a cluster that can be safely deleted when your evaluation is complete.

1. Follow instructions for the [installation prerequisites](./install/INSTALL_PREREQ.md)
2. Install using xip.io DNS or OCI DNS below as describe next
3. Verify installation

## Install Verrazzano using xip.io
Run the following scripts in order:

```
   ./install/1-install-istio.sh
   ./install/2a-install-system-components-magicdns.sh
   ./install/3-install-verrazzano.sh
   ./install/4-install-keycloak.sh
```
**OR**
##  Install using OCI DNS

Installing Verrazzano on OCI DNS requires the following environment variables needed to create OCI DNS records:

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

Run the following scripts in order:
```
   ./install/1-install-istio.sh
   ./install/2a-install-system-components-ocidns.sh -n <env-name> -s 
   ./install/3-install-verrazzano.sh -n <env-name> -d oci -s <oci-dns-zone-name>
   ./install/4-install-keycloak.sh -n <env-name> -d oci -s <oci-dns-zone-name>
```

## Getting the console URLs


## Getting Credentials
You will need the credentials to access the various consoles installed by Verrazzano.

### Consoles accessed by the same username/password.

- UI Console
- Grafana
- Prometheus
- Kibana
- Elasticsearch

User:  `verrazzano`

You can get the password by running the following command:  
`kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo`

### The Keycloak Admin console:

User `keycloakadmin`
 
run this command to get the password:  
kubectl get secret --namespace keycloak keycloak-http -o jsonpath={.data.password} | base64 --decode; echo

### The Rancher console:

User `keycloakadmin`
 
run this command to get the password:  
`kubectl get secret --namespace cattle-system rancher-admin-secret -o jsonpath=“{.data.password}” | base64 --decode; echo`




## More Information
More detailed installation instructions can be found in the `install` folder in this repository.

Example applications can be found in the `examples` folder.

For additional information, see the [Verrazzano documentation](https://verrazzano.io/doc).
