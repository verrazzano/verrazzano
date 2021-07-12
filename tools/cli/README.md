# Verrazzano CLI

The Verrazzano Command Line Interface (CLI) is a tool which can be used to interact with the Verrazzano resources.

## Building the CLI

The binary will be located in ``/bin``.

Run ``make cli``

## Usage

Run ``vz --help`` for a list of available commands.

````
# Login into the Verrazzano API server
$ vz login  https://verrazzano.example.com
Login sucessful!

# List the clusters
$ vz cluster list
Name    Age     Status  Description
local   2d9m    Ready   Admin Cluster

# Register a managed cluster
$ vz cluster register managed1 -d "Managed Cluster" -c "ca-secret-managed1"
verrazzanomanagedcluster/managed1 created

# Fetch the manifest to be applied on the managed cluster to complete registration
$ vz cluster get-registration-manifest managed1 > managed1.yaml | kubectl --context kind-managed1 apply -f .
$ vz cluster list
Name        Age     Status  Description
local       2d9m    Ready   Admin Cluster
managed1    30s     Ready   Managed Cluster

# Create a new project
$ vz project add project1 -p "managed1"
verrazzanoproject/project1 created

# List all the projects
$ vz project list
Name        Age     Clusters    Namespaces
project1    30s     managed1    project1

# Delete the project
$ vz project delete project1
verrazzanoproject/project1 deleted

# Deregister a managed cluster
$ vz cluster deregister managed1
verrazzanomanagedcluster/managed1 deregistered

# Logout of the API server
$ vz logout
Logout successful!
````
