# Summary
Analysis detected that there were pods which had issues due to failures to pull an image or images where the root cause was that the service was not available.

The service might be unreachable or might be incorrect.

## Steps
1. Review the analysis data; it enumerates the pods and related messages about which images had this issue.
2. Confirm that the registry for the image is correct.
3. The messages might identify a connectivity issue.
4. If the service is experiencing an outage, then consult the specific service status page. For common service status pages, see [Related information](#related-information).

## Related information
* [GitHub Status](https://www.githubstatus.com/)
* [OCI Status](https://ocistatus.oraclecloud.com/)
* [Kubernetes Troubleshooting](https://kubernetes.io/docs/tasks/debug-application-cluster/troubleshooting/)
