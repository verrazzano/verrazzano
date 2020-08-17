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

All resources (applications, pods, etc.) that are not directly installed by the Verrazzano installation should be removed from the cluster.
The uninstall scripts work by scraping the cluster and looking for resources by way of pattern matching.
Any resource left on the cluster is at risk of being deleted by the uninstall scripts, so they should be preemptively removed.

## 2. Uninstall Verrazzano

> **NOTE:** All references to files are in relation to the project directory

### Partial Uninstall

Each script in the `uninstall` directory correlates with a counterpart in the `install` directory.
For example, the resources created in
```
    ./install/1-install-istio.sh
```
can be removed with
```
    ./uninstall/1-uninstall-istio.sh
```
The formatting of these commands can be applied to all of the install scripts.

### Complete Uninstall

To completely uninstall all Verrazzano components, These scripts need to execute:
```
   ./uninstall/1-uninstall-istio.sh
   ./uninstall/2-uninstall-system-components-ocidns.sh
   ./uninstall/3-uninstall-verrazzano.sh
   ./uninstall/4-uninstall-keycloak.sh
```
After these processes, the cluster should return to its original state.

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