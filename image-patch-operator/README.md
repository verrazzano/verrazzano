# Image Patch Operator
This document goes through the steps of building, deploying, and testing the image patch operator.

### Build the Required Docker Images
First, build the Docker image for the image patch operator.
```bash
cd <path-to-verrazzano-repo>/image-patch-operator
make docker-build
```
Build the Docker image for the WebLogic Image Tool.
```bash
cd <path-to-verrazzano-repo>/image-patch-operator/weblogic-imagetool
make docker-build
```
Next, verify that the two images have been created. This will show the names and tags for each image.
```bash
docker images
```
At this point, load these two images into your Kubernetes cluster. For example, if you are using a minikube cluster, run the following commands.
```bash
minikube image load <image-patch-operator-name>:<image-patch-operator-tag>
minikube image load <image-tool-name>:<image-tool-tag>
```

### Install the Helm Chart
```bash
# Defining this environment variable is just for convenience.
export IPO_CHART=<path-to-verrazzano-repo>/image-patch-operator/helm_config/charts/image-patch-operator

# Performs the install.
helm install verrazzano-image-patch-operator $IPO_CHART --create-namespace --namespace verrazzano-system --set-string imagePatchOperator.image=<image-patch-operator-name>:<image-patch-operator-tag> --set-string imageTool.image=<image-tool-name>:<image-tool-tag>
```
After installing the Helm Chart, the ImageBuildRequest custom resource definition should be defined, which can be verified with `kubectl get crd`. The pod for the image patch operator should also running, which can be seen with `kubectl get pods -n verrazzano-system`.

### Create a Secret with Credentials for Pushing the Image
The Secret can be manually created using `kubectl`.<br>
First, create three local files containing your credentials and the registry you would like to push the WebLogic ImageTool image to.
```bash
echo <your-username> > username.txt
echo <your-password> > password.txt
echo <container-registry-name> > registry.txt
```
The following command will create the Secret.
```bash
kubectl create secret generic verrazzano-imagetool -n verrazzano-system \
  --from-file=username=./username.txt \
  --from-file=password=./password.txt \
  --from-file=registry=./registry.txt
```
