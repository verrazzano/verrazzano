## WebLogic Application

Sample WebLogic application, for testing the VerrazzanoWebLogicWorkload.

## Requires

[Maven](https://maven.apache.org/download.cgi)

## Steps to build the application
The bash script setup/build.sh creates the auxiliary image for model in image deployment, by including the sample application under wlsdeploy/applications.

    $ cd <application root directory>
    $ mvn clean package
    $ cd setup; ./build.sh <container registry>/<image>:<version>

The application doesn't validate any feature, it just includes a welcome page.

Copyright (c) 2022, Oracle and/or its affiliates.
