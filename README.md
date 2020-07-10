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

## tl;dr
To install Verrazzano, follow these steps:  
1. Create an OKE cluster  
2. Start OCI Cloud Shell  
3. Clone this repo in the Cloud Shell home.
4. Run the following scripts:  
   - `export CLUSTER_TYPE=OKE`
   - `export VERRAZZANO_KUBECONFIG=<path to valid kubernetes config>`
   - `export KUBECONFIG=<path to valid kubernetes config>`
   - `kubectl create secret docker-registry ocr --docker-username=<username> --docker-password=<password> --docker-server=container-registry.oracle.com`
   - `./install/1-install-istio.sh`
   - `./install/2a-install-system-components-magicdns.sh`
   - `./install/3-install-verrazzano.sh`
   - `./install/4-install-keycloak.sh`
5. (Optional) Install some example applications - see below for details.

> **NOTE**: This alpha release of Verrazzano is intended for installation in a single OKE or Kind cluster. You should only install Verazzano in a cluster that can be safely deleted when your evaluation is complete.

## Deploying the example applications

To deploy the example applications, please see the following instructions:

* [Bob's Books](./examples/bobs-books/README.md)
* [Helidon Hello World](./examples/hello-helidon/README.md)
* TBD

## More Information

For additional information, see the [Verrazzano documentation](https://verrazzano.io/doc).

More detailed [installation instructions](./install/README.md) can be found in the `install` directory.

## Contributing to Verrazzano

Oracle welcomes contributions to this project from anyone.  Contributions may be reporting an issue with the operator or submitting a pull request.  Before embarking on significant development that may result in a large pull request, it is recommended that you create an issue and discuss the proposed changes with the existing developers first.

If you want to submit a pull request to fix a bug or enhance an existing feature, please first open an issue and link to that issue when you submit your pull request.

If you have any questions about a possible submission, feel free to open an issue too.

## Contributing to the Verrazzano repository

Pull requests can be made under The Oracle Contributor Agreement (OCA), which is available at [https://www.oracle.com/technetwork/community/oca-486395.html](https://www.oracle.com/technetwork/community/oca-486395.html).

For pull requests to be accepted, the bottom of the commit message must have the following line, using the contributor’s name and e-mail address as it appears in the OCA Signatories list.

```
Signed-off-by: Your Name <you@example.org>
```

This can be automatically added to pull requests by committing with:

```
git commit --signoff
```

Only pull requests from committers that can be verified as having signed the OCA can be accepted.

## Pull request process

*	Fork the repository.
*	Create a branch in your fork to implement the changes. We recommend using the issue number as part of your branch name, for example, `1234-fixes`.
*	Ensure that any documentation is updated with the changes that are required by your fix.
*	Ensure that any samples are updated if the base image has been changed.
*	Submit the pull request. Do not leave the pull request blank. Explain exactly what your changes are meant to do and provide simple steps on how to validate your changes. Ensure that you reference the issue you created as well. We will assign the pull request to 2-3 people for review before it is merged.

## Introducing a new dependency

Please be aware that pull requests that seek to introduce a new dependency will be subject to additional review.  In general, contributors should avoid dependencies with incompatible licenses, and should try to use recent versions of dependencies.  Standard security vulnerability checklists will be consulted before accepting a new dependency.  Dependencies on closed-source code, including WebLogic Server, will most likely be rejected.

