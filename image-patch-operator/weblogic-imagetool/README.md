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

# Add the installers needed to create the image
imagetool cache addInstaller --type wls --version 12.2.1.4.0 --path ./installers/fmw_12.2.1.4.0_wls.jar
imagetool cache addInstaller --type jdk --version 8u281  --path ./installers/jdk-8u281-linux-x64.tar.gz
imagetool cache addInstaller --type wdt --version latest --path ./installers/weblogic-deploy.zip

# Creates the image
imagetool create --tag testimage:1 --builder podman --jdkVersion 8u281 --version 12.2.1.4.0 --dryRun
```
Because of the `--dryRun` flag that was passed in, the Dockerfile for the new image is dumped to standard output, but no image is actually created.

Omit the `--dryRun` option to actually create an image.
```bash
imagetool create --tag testimage:1 --builder podman --jdkVersion 8u281 --version 12.2.1.4.0
```
After creating an image this way, we can verify that it exists by running `podman images`, whose output should include an image called "testimage" with a tag of 1.


