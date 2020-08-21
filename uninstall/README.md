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

## 1. Prepare for Uninstall

All resources (applications, pods, etc.) created and/or managed by Verrazzano should be checked before uninstalling.
The uninstall script will delete all Verrazzano resources including Verrazzano-managed applications. 
Please make sure that you want to delete these applications before uninstalling Verrazzano.
If so, the uninstall script will take care of deleting all applications deployed with Verrazzano.

## 2. Uninstall Verrazzano

> **NOTE:** All references to files are in relation to the project directory

### Complete Uninstall

To completely uninstall all Verrazzano components, This script needs to be executed:
```
   ./uninstall/uninstall-verrazzano.sh
```
After these processes, the cluster should return to its original state. 
Verrazzano should be deleted in its entirety to avoid unwanted complications.

## 3. Verify uninstall
A complete verification of an uninstall of Verrazzano would require a complete scrape of the resources located on the cluster.
A quick verification could include checking that only the default namespaces are present:
```
    NAME                 STATUS   AGE
    default              Active   1d
    kube-node-lease      Active   1d
    kube-public          Active   1d
    kube-system          Active   1d
```

## More Information
For additional information, see the [Verrazzano documentation](https://verrazzano.io/docs).