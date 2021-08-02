# Image Patch Operator
This document describes building, deploying, and testing the image patch operator.

### Build the Required Docker Images
First, build the Docker image for the image patch operator.
```bash
$ make docker-build
```
Build the Docker image for the WebLogic Image Tool.
```bash
$ cd weblogic-imagetool
$ make docker-build
```
Next, verify that the two images have been created. The following command will display the names and tags for each image.
```bash
$ docker images
```
Your output will be similar to the following:
```plaintext
REPOSITORY                            TAG              IMAGE ID       CREATED         SIZE
verrazzano-weblogic-image-tool-dev    local-00bf9bd7   d975a2f4b85a   4 minutes ago   1.64GB
verrazzano-image-patch-operator-dev   local-00bf9bd7   a0865a3d3e16   5 days ago      187MB
```
Now, create a Kubernetes cluster, and load these two images into your cluster. For example, if you are using a Kind cluster, run the following commands.

Make sure you fill out the hostPath field with a local path to a directory containing the installers for the JDK, WLS, and Weblogic Image Tool.
If you don't already have these installers, you may download them locally using the links below
- [JDK](https://www.oracle.com/java/technologies/javase-jdk16-downloads.html)
- [WLS](https://www.oracle.com/middleware/technologies/weblogic-server-downloads.html)
- [WIT](https://github.com/oracle/weblogic-image-tool/releases)

Additionally, please add the JDK installer to the image-patch-operator/weblogic-imagetool/installers directory.

```bash
# Create the cluster
$ kind create cluster --config - <<EOF
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
    extraMounts:
      - hostPath: <path-to-installers>
        containerPath: /installers
EOF

# Load the images
$ kind load docker-image --name kind <operator-image-name>:<operator-image-tag>
$ kind load docker-image --name kind <imagetool-image-name>:<imagetool-image-tag>
```

### Install the Helm Chart
```bash
# This command assumes that your current working directory is the directory containing this README.
# Defining this environment variable is just for convenience.
$ export IPO_CHART=$(pwd)/helm_config/charts/image-patch-operator

# Perform the install.
$ helm install verrazzano-image-patch-operator $IPO_CHART --create-namespace --namespace verrazzano-system --set-string imagePatchOperator.image=<operator-image-name>:<operator-image-tag> --set-string imageTool.image=<imagetool-image-name>:<imagetool-image-tag>
```
Verify that the ImageBuildRequest custom resource definition has been created.
```bash
$ kubectl get crd imagebuildrequests.images.verrazzano.io
```
This will show an output similar to the following:
```plaintext
NAME                                      CREATED AT
imagebuildrequests.images.verrazzano.io   2021-07-19T15:32:26Z
```
Verify that the image patch operator is running.
```bash
$ kubectl get pods -n verrazzano-system
```
This will show an output similar to the following:
```plaintext
NAME                                               READY   STATUS         RESTARTS   AGE
verrazzano-image-patch-operator-7477f65ccf-rjmsc   1/1     Running        0          6m42s
```

### Create a Secret with Credentials for Pushing the Image
The Secret can be manually created using `kubectl`.<br>
```bash
$ kubectl create secret generic verrazzano-imagetool -n verrazzano-system \
  --from-literal=username=<your-username> \
  --from-literal=password=<your-password> \
  --from-literal=registry=<container-registry-name>
```
Verify that the secret has been created.
```bash
$ kubectl get secret -n verrazzano-system
```
The output will be similar to the following:
```plaintext
NAME                                                    TYPE                                  DATA   AGE
...                                                     ...                                   ...    ...
verrazzano-imagetool                                    Opaque                                3      4s
```

### Create the ImageBuildRequest
The following command will create the ImageBuildRequest.
```bash
$ kubectl apply -f - <<-EOF
apiVersion: images.verrazzano.io/v1alpha1
kind: ImageBuildRequest
metadata:
  name: cluster1
  namespace: verrazzano-system
spec:
  baseImage: ghcr.io/oracle/oraclelinux:8-slim
  jdkInstaller: jdk-8u281-linux-x64.tar.gz
  webLogicInstaller: fmw_12.2.1.4.0_wls.jar
  jdkInstallerVersion: 8u281
  webLogicInstallerVersion: 12.2.1.4.0
  image:
    name: <your-image-name>
    tag: <your-image-tag>
    registry: <container-registry>
    repository: <your-repository>
EOF
```
Verify the status of the ImageBuildRequest.
```bash
$ kubectl get ImageBuildRequest -A
```
This will show an output similar to the below block. The `STATUS` section will update accordingly.
```plaintext
NAMESPACE           NAME       STATUS
verrazzano-system   cluster1   BuildStarted
```

### Check the Status of the Image
After creating the ImageBuildRequest, the image created by the WebLogic Image Tool will be in the process of building and being pushed.
To track this progress, find the name of the pod that was created as a result of the ImageBuildRequest.
```bash
$ kubectl get pods -n verrazzano-system
```
This will show a pod with a name in the format of `verrazzano-images-cluster1-XXXXX`.<br>
To track the progress of your WebLogic Image Tool image, check the logs of this pod.
```bash
$ kubectl logs -f -n verrazzano-system verrazzano-images-cluster1-XXXXX
```

### Troubleshooting
After creating an ImageBuildRequest, if you encounter problems with the building of your image, you may be able to help diagnose your problem through the use of the WebLogic Image Tool Dry Run feature. Running an ImageJob as a Dry Run will result in the Dockerfile of the image to be printed to stdout - the image will not be built or pushed.

There are two different methods to run an ImageJob as a Dry Run. The first is to edit the Deployment that gets created by the helm install. By default, the "IBR_DRY_RUN" environmental variable of the Deployment will be set to false. You may edit the Deployment and set this variable to true. This will run all future ImageJobs as a Dry Run (you may set this variable back to false to run ImageJobs regularly).

The second method you may use, is to edit the values.yaml file in the helm_config directory. The values.yaml file lists a variable "dryRun" which by default is set to false. Setting this variable to true, and then preforming a helm install will result in future ImageJobs to run as Dry Runs. You may change back to running ImageJobs normally by editing the Deployment and setting the "IBR_DRY_RUN" variable back to false.

To view the output of an ImageJob Dry Run, follow the steps located in the "Check the Status of the Image" section of this document.