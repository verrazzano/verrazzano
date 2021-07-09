# WIT in Docker Image

This Dockerfile is used to run WebLogic Image Tool inside a Docker container. <br>
Docker Desktop is needed to run the below example.

## Create a Image using WIT inside a Docker container
This example shows how to use this Dockerfile to create a local container that supports the WebLogic Image Tool.

First, build the image from the Dockerfile.
```bash
# This command assumes that your current working directory is the directory that contains this README.
make docker-build
```
Start the container with the following command. Replace `image-id` with the ID corresponding to the newly created `verrazzano-weblogic-image-tool-dev` image, which you can view by using the `docker images` command. This will start an interactive shell prompt inside the Docker container.
```bash
docker run -it --privileged --entrypoint /bin/bash image-id
```

While inside the container's shell prompt, use the imagetool to create a sample image.
```bash
# Creates an alias for the `imagetool.sh` shell script for convenience
alias imagetool=/home/verrazzano/imagetool/bin/imagetool.sh

imagetool create --tag testimage:1 --builder podman --jdkVersion 8u281 --version 12.2.1.4.0 --dryRun
```
Because of the `--dryRun` flag that was passed in, the Dockerfile for the new image is dumped to standard output, but no image is actually created.

Omit the `--dryRun` option to actually create an image.
```bash
imagetool create --tag testimage:1 --builder podman --jdkVersion 8u281 --version 12.2.1.4.0
```
After creating an image this way, we can verify that it exists by running `podman images`, whose output should include an image called "testimage" with a tag of 1.


As an alternative to entering the container's shell prompt, the same `imagetool create` commands can be used as inputs to `docker run`. The following commands should result in a similar output to the above instructions.
```bash
# This command assumes that your current working directory is the directory that contains this README.
make docker-build

docker run --privileged image-id create --tag testimage:1 --builder podman --jdkVersion 8u281 --version 12.2.1.4.0 --dryRun

# Without --dryRun, this actually creates the image, but the container shuts down afterward anyway.
docker run --privileged image-id create --tag testimage:1 --builder podman --jdkVersion 8u281 --version 12.2.1.4.0
```

