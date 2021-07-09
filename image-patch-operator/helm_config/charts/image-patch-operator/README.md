# Helm Chart for Image Patch Operator

This Helm chart installs the image patch service components into a Verrazzano environment.

## Example: Install Image Patch Operator into Local Minikube Cluster
First, build the Docker images locally for the Image Patch Operator and WebLogic Image Tool.
```bash
# This assumes your starting directory is the root directory of the verrazzano repository.
cd image-patch-operator
make docker-build
cd weblogic-imagetool
make docker-build
```
Check that the two images are built locally by running `docker images`. This will also show the name and tag for each image.

Next, start up a minikube cluster.
```bash
minikube start
```
Once the cluster is created, load the two Docker images into the cluster. Replace the image names and tags according to your `docker images` output.
```bash
minikube image load <image-patch-operator-name>:<image-patch-operator-tag>
minikube image load <image-tool-name>:<image-tool-tag>
```

Install this helm chart.
```bash
# This assumes your current working directory is the root directory of the verrazzano repository.
# Defining this environment variable is just for convenience.
export IPO_CHART=$(pwd)/image-patch-operator/helm_config/charts/image-patch-operator

# Performs the install.
helm install verrazzano-image-patch-operator $IPO_CHART --create-namespace --namespace verrazzano-system --set-string imagePatchOperator.image=<image-patch-operator-name>:<image-patch-operator-tag> --set-string imageTool.image=<image-tool-name>:<image-tool-tag>
```

Verify that the Image Patch Operator pod is running and that the ImageBuildRequest custom resource definition is applied.
```bash
kubectl get pods -n verrazzano-system
kubectl get crd
```

To uninstall this Helm chart from the cluster, run the following command.
```bash
helm uninstall verrazzano-image-patch-operator -n verrazzano-system
```

