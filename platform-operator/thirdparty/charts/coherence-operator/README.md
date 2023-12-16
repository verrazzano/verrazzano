<!--
  Copyright 2020, Oracle Corporation and/or its affiliates.
  Licensed under the Universal Permissive License v 1.0 as shown at
  http://oss.oracle.com/licenses/upl.
-->

# coherence-operator
Install coherence-operator to work with Coherence clusters on Kubernetes.

## Introduction

This chart install a coherence-operator deployment on a 
[Kubernetes](https://kubernetes.io) cluster using the [Helm](https://helm.sh)
package manager.

## Prerequisites
* Kubernetes 1.13 or above
* Helm 3 or above

## Installing the Chart
To install the chart with the release name `sample-coherence-operator`:

```
@ helm install sample-coherence-operator coherence-operator
```

The command deploys coherence-operator on the Kubernetes cluster in the
default configuration. The [configuration](#configuration) section list
parameters that can be configured during installation.

## Uninstalling the Chart
To uninstall the `sample-coherence-operator` deployment:

```
$ helm delete sample-coherence-operator
```

