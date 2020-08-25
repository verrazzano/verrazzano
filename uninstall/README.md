# Uninstall

Verrazzano creates a list of services and resources that reside on the cluster in which Verrazzano is installed.
If these resources need to be deleted, the scripts in the `uninstall` directory will make that process easier. At any stage of the installation or development process, you can use these scripts to clean the cluster of all current Verrazzano resources.

## Software Requirements

The software requirements for the [install](../install/README.md) are also required for the uninstall:
* curl
* helm
* jq
* kubectl
* openssl
* patch (for OCI DNS installation)

## Uninstall Verrazzano

> **NOTE:** All references to files are in relation to the project directory.

The uninstall script will delete all Verrazzano resources including any Verrazzano-managed applications.

To completely uninstall all Verrazzano components, run:
```
   ./uninstall/uninstall-verrazzano.sh
```
