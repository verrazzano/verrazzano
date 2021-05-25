# Installing Verrazzano From a Private Registry

This document details the steps required to populate a private image registry with Verrazzano images and install Verrazzano from that registry.

## Prerequisites
You must have the following software installed:

 - [Docker](https://docs.docker.com/get-docker/)
 - [kubectl](https://kubernetes.io/docs/tasks/tools/)
 - [Helm](https://helm.sh/docs/intro/install/) (version 3.x+)
 - [jq](https://github.com/stedolan/jq/wiki/Installation)

## Load the Images
Before running the script that pushes images to your private registry, execute `docker login [SERVER]` with your credentials.

Run the included helper script to push images to the registry. Replace the registry and repository values with your own.
```
$ sh vz-registry-image-helper.sh -t myreg.io -r myrepo/v8o -l .
```
Although most images can be protected using credentials stored in an image pull secret, the following images **must** be public:

All of the Rancher images.
```
$ cat verrazzano-bom.json | jq -r '.components[] |
    select(.name == "rancher")'
```

The Verrazzano Platform Operator image. See the next section for details on how to determine the image name and tag.

## Install Verrazzano
If your registry is configured to require credentials in order to pull images, create the following secret in the default namespace. Note that the name of the secret is important.
```
$ kubectl create secret docker-registry verrazzano-container-registry \  
	--docker-server=myreg.io --docker-username=myreguser \  
	--docker-password=xxxxxxxx --docker-email=me@example.com
```
Locate the Verrazzano Platform Operator repository in the bill of materials file.
```
$ cat verrazzano-bom.json | jq -r '.components[] |
    select(.name == "verrazzano-platform-operator") |
    .subcomponents[0].repository'
```
Locate the Verrazzano Platform Operator image and tag in the bill of materials file.
```
$ cat verrazzano-bom.json | jq -r '.components[] |
    select(.name == "verrazzano-platform-operator") | .subcomponents[] |
    select(.name == "verrazzano-platform-operator") | 
    .images[0].image,.images[0].tag'
```
Using those repository, image, and tag values, run the `helm` command to install the Verrazzano platform operator.
```
$ helm upgrade --install myv8o ./charts/verrazzano-platform-operator \  
	--set image=myreg.io/myrepo/v8o/<repository>/<image>:<tag> \  
	--set global.registry=myreg.io --set global.repository=myrepo/v8o
```
Wait for the operator deployment to complete.
```
$ kubectl -n verrazzano-install rollout status deployment/verrazzano-platform-operator
```
Once the Verrazzano Platform Operator is running, proceed with installing Verrazzano as documented in https://verrazzano.io/docs/setup/install/installation/.
