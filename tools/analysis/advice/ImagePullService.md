# Summary
Analysis detected that there were pods which had issues due to failures to pull an image/images where the root cause was the service wasn't available.

The service may be unreachable or may be incorrect.

# Steps
* Review the analysis data supplied. This will enumerate the pods and related messages which provide the images which had this issue.
* Confirm that the registry for the image is correct
* The messages may identify a connectivity issue
* The service may be experiencing an outage, consult the specific service status page (see related information for some common service status page links)

# Related Information
* https://www.githubstatus.com/
* https://ocistatus.oraclecloud.com/
* https://kubernetes.io/docs/tasks/debug-application-cluster/troubleshooting/
