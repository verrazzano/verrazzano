# Uninstall

**IMPORTANT NOTE:** This uninstaller is provided as an EXPERIMENTAL feature.

You can completely uninstall Verrazzano and any Verrazzano-managed applications from your cluster.  You must delete all Verrazzano application models and Verrazzano application bindings from your system
before uninstalling Verrazzano.  The uninstaller will list any deployed models and bindings and prompt for whether you want them to be deleted before proceeding.

## Software Requirements

The software requirements for the [install](../install/README.md) are also required for the uninstall:
* curl
* helm
* jq
* kubectl
* openssl
* patch (for OCI DNS installation)

## Uninstall Verrazzano

* Set the following `ENV` vars:
```
   export CLUSTER_TYPE=<OKE|KIND|OLCNE>
   export VERRAZZANO_KUBECONFIG=<path to valid kubernetes config>
   export KUBECONFIG=$VERRAZZANO_KUBECONFIG
```
*  To completely uninstall all Verrazzano components including any Verrazzano-managed applications, run:
```
   ./uninstall-verrazzano.sh
```
