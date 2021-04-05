# Summary
Analysis detected that there were pods which had issues due to failures to pull an image/images where the root cause was that the image wasn't found.

# Steps
* Review the analysis data supplied. This will enumerate the pods and related messages which provide the images which had this issue.
* Confirm that the image name, digest, and tag are correctly specified 

# Related Information
* https://kubernetes.io/docs/tasks/debug-application-cluster/troubleshooting/
