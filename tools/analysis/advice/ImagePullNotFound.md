# Summary
Analysis detected that there were pods which had issues due to failures to pull an image or images where the root cause was that the image wasn't found.

# Steps
1. Review the analysis data supplied. It enumerates the pods and related messages about which images had this issue.
2. Confirm that the image name, digest, and tag are correctly specified.

# Related Information
* https://kubernetes.io/docs/tasks/debug-application-cluster/troubleshooting/
