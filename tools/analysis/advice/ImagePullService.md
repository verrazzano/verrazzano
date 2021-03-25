# Summary
Analysis detected that there were pods which had issues due to failures to pull an image/images where the root cause was the service wasn't available.

The service may be unreachable or may be incorrect.

# Steps
* Review the analysis data supplied. This will enumerate the pods and related messages which provide the images which had this issue.
* Confirm that the registry for the image is correct
* The messages may identify a connectivity issue

# Related Information
* https://kubernetes.io/docs/tasks/debug-application-cluster/troubleshooting/
