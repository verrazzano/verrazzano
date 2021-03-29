# Summary
Analysis detected that there were nodes reporting insufficient memory.

# Steps
* Review the analysis data supplied to identify the specific nodes involved
* Review the nodes to determine why they do not have sufficient memory
  * Are the nodes sized correctly for the workload?
    * Refer to https://verrazzano.io/docs/setup/install/installation/ for minimum resources required for installing Verrazzano
    * Refer to documentation for other applications that you are deploying for resource guidelines and take those into account
  * Is there something else unexpected executing on the nodes or consuming more memory than expected?

# Related Information
* https://verrazzano.io/docs/setup/install/installation/
* https://kubernetes.io/docs/tasks/debug-application-cluster/troubleshooting/
