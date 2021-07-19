# Image Patch Operator
This document goes through the steps of building, deploying, and testing the image patch operator.

### Build the Required Docker Images
First, build the Docker image for the image patch operator.
```bash
make docker-build
```
Build the Docker image for the WebLogic Image Tool.
```bash
cd weblogic-imagetool
make docker-build
```
Next, verify that the two images have been created. This will show the names and tags for each image.
```bash
docker images
```
Your output should be similar to
```plaintext
REPOSITORY                            TAG              IMAGE ID       CREATED         SIZE
verrazzano-weblogic-image-tool-dev    local-00bf9bd7   d975a2f4b85a   4 minutes ago   1.64GB
verrazzano-image-patch-operator-dev   local-00bf9bd7   a0865a3d3e16   5 days ago      187MB
```
At this point, create a Kubernetes cluster, and load these two images into your cluster. For example, if you are using a Kind cluster, run the following commands.
```bash
# Create the cluster
kind create cluster --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: ClusterConfiguration
        apiServer:
          extraArgs:
            "service-account-issuer": "kubernetes.default.svc"
            "service-account-signing-key-file": "/etc/kubernetes/pki/sa.key"
EOF

# Load the images
kind load docker-image --name kind <operator-image-name>:<operator-image-tag>
kind load docker-image --name kind <imagetool-image-name>:<imagetool-image-tag>
```

### Install the Helm Chart
```bash
# This command assumes that your current working directory is the directory containing this README.
# Defining this environment variable is just for convenience.
export IPO_CHART=$(pwd)/helm_config/charts/image-patch-operator

# Performs the install.
helm install verrazzano-image-patch-operator $IPO_CHART --create-namespace --namespace verrazzano-system --set-string imagePatchOperator.image=<operator-image-name>:<operator-image-tag> --set-string imageTool.image=<imagetool-image-name>:<imagetool-image-tag>
```
Verify that the ImageBuildRequest custom resource definition has been created.
```bash
kubectl get crd imagebuildrequests.images.verrazzano.io
```
This should show an output similar to
```plaintext
NAME                                      CREATED AT
imagebuildrequests.images.verrazzano.io   2021-07-19T15:32:26Z
```
Verify that the image patch operator is running.
```bash
kubectl get pods -n verrazzano-system
```
This should show an output similar to
```plaintext
NAME                                               READY   STATUS         RESTARTS   AGE
verrazzano-image-patch-operator-7477f65ccf-rjmsc   1/1     Running        0          6m42s
```

### Create a Secret with Credentials for Pushing the Image
The Secret can be manually created using `kubectl`.<br>
```bash
kubectl create secret generic verrazzano-imagetool -n verrazzano-system \
  --from-literal=username=<your-username> \
  --from-literal=password=<your-password> \
  --from-literal=registry=<container-registry-name>
```
