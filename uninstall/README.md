# Uninstall

To uninstall Verrazzano, use the scripts in the `uninstall` directory.

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
