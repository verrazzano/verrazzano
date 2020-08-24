# Uninstall

Verrazzano creates a list of services and resources that reside on the cluster in which Verrazzano is installed.
In the case that these resources need to be deleted, the `uninstall` directory of Verrazzano should make that process easier.
At any stage of the installation or development process, these scripts can be used to clean the cluster of all current Verrazzano resources.

## Software Requirements

The software requirements for the [install](../install/README.md) are also required for the uninstall:
* curl
* helm
* jq
* kubectl
* openssl
* patch (for OCI DNS installation)

## Uninstall Verrazzano

> **NOTE:** All references to files are in relation to the project directory

The uninstall script will delete all Verrazzano resources including any Verrazzano-managed applications.

To completely uninstall all Verrazzano components, this script needs to be executed:
```
   ./uninstall/uninstall-verrazzano.sh
```

## More Information
For additional information, see the [Verrazzano documentation](https://verrazzano.io/docs).