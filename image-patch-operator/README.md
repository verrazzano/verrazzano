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
```
REPOSITORY                            TAG              IMAGE ID       CREATED             SIZE
verrazzano-weblogic-image-tool-dev    local-00c6bd61   c3a7bfc12230   About an hour ago   1.64GB
verrazzano-image-patch-operator-dev   local-d86959a5   a0865a3d3e16   5 days ago          187MB
```
At this point, create a Kubernetes cluster, and load these two images into your cluster. For example, if you are using a Kind cluster, run the following commands.
```bash
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
```
kind load docker-image --name kind verrazzano-image-patch-operator-dev:local-2b27f37c
kind load docker-image --name kind verrazzano-weblogic-image-tool-dev:local-d86959a5
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
```bash
kubectl create secret generic verrazzano-imagetool -n verrazzano-system \
  --from-literal=username=<your-username> \
  --from-literal=password=<your-password> \
  --from-literal=registry=<container-registry-name>
```
