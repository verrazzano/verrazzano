# Summary
Analysis detected that there were nodes reporting insufficient memory.

## Steps
1. Review the analysis data to identify the specific nodes involved.
2. Review the nodes to determine why they do not have sufficient memory.

   a. Are the nodes sized correctly for the workload?

      1. For the minimum resources required for installing Verrazzano, see https://verrazzano.io/docs/setup/install/installation/.
      2. Refer to documentation for other applications that you are deploying for resource guidelines and take those into account.

  b. Is something unexpected running on the nodes or consuming more memory than expected?

## Related information
* https://verrazzano.io/docs/setup/install/installation/
* https://kubernetes.io/docs/tasks/debug-application-cluster/troubleshooting/
