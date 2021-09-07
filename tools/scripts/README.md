# Installing Verrazzano From a Private Registry

Follow these required steps to populate a private image registry with Verrazzano images and install Verrazzano from that registry.

## Prerequisites
You must have the following software installed:

 - [Docker](https://docs.docker.com/get-docker/)
 - [kubectl](https://kubernetes.io/docs/tasks/tools/)
 - [Helm](https://helm.sh/docs/intro/install/) (version 3.x+)
 - [jq](https://github.com/stedolan/jq/wiki/Installation)

## Loading the images

There are two options for loading Docker images into your private registry using the provided `vz-registry-image-helper.sh` script:

1. To pull Verrazzano Docker images from their default public registries, tag them, and push them to your private registry. 
2. To load downloaded Verrazzano Docker image tar files into the local registry, tag them, amd push them to your private registry.

Option (1) requires that you have downloaded the minimal Verrazzano Zip file distribution.

Option (2) requires that you have downloaded the Zip file containing the full set of Verrazzano Docker images exported
as tar files.

Before running the `vz-registry-image-helper.sh` script you must run `docker login [SERVER]` with your credentials for
your private registry, where `[SERVER]` is the DNS name of your private registry.

For use with the examples in this document, define the following variables with respect to your target registry and repository:

* `MYREG`
* `MYREPO`
* `VPO_IMAGE`

These identify the target Docker registry and repository, and the Verrazzano Platform Operator image, as defined in the BOM file.

For example, using a target registry of `myreg.io` and a target repository of `myrepo/v8o`:

```
MYREG=myreg.io
MYREPO=myrepo/v8o
VPO_IMAGE=$(cat verrazzano-bom.json | jq -r '.components[].subcomponents[] | select(.name == "verrazzano-platform-operator") | "\(.repository)/\(.images[].image):\(.images[].tag)"')
```

### Option 1: Pull, tag, and push Verrazzano images from public repositories

Go to the directory where you extracted the images archive and run the included helper script to pull, tag, and push images to the registry:

```
$ sh vz-registry-image-helper.sh -t $MYREG -r $MYREPO
```

### Option 2: Load, tag, and push local Verrazzano images

Go to the directory where you extracted the images archive and run the included helper script to pull, tag, and push images to the registry:

```
$ sh vz-registry-image-helper.sh -t $MYREG -r $MYREPO -l .
```

## Pre-install

Although most images can be protected using credentials stored in an image pull secret, the following images **must** be public:

* All of the Rancher images in the `rancher/additional-rancher` subcomponent.
    ```
    $ cat verrazzano-bom.json | jq -r '.components[].subcomponents[] | select(.name == "additional-rancher") | .images[] | "\(.image):\(.tag)"'
    ```
* The Verrazzano Platform Operator image identified by `$VPO_IMAGE`, as defined above.

For all other Verrazzano Docker images in the private registry that are not explicitly marked public, you will need to 
create the secret `verrazzano-container-registry` in the `default` namespace, with the appropriate credentials for the 
registry, identified by `$MYREG`.

For example,

```
$ kubectl create secret docker-registry verrazzano-container-registry \  
	--docker-server=$MYREG --docker-username=myreguser \  
	--docker-password=xxxxxxxx --docker-email=me@example.com
```

## Install Verrazzano

Next, install the Verrazzano Platform Operator using the image defined by `$MYREG/$MYREPO/$VPO_IMAGE`.  

```
helm upgrade --install myv8o ./charts/verrazzano-platform-operator \
    --set image=${MYREG}/${MYREPO}/${VPO_IMAGE} --set global.registry=${MYREG} \
    --set global.repository=${MYREPO} --wait
```

After the Verrazzano Platform Operator is running, proceed with installing Verrazzano as documented at 
https://verrazzano.io/docs/setup/install/installation/.
