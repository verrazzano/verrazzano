# Summary
Analysis detected that there were pods which had issues due to failures to pull an image or images.

The root cause was `rate limit exceeded` errors while pulling images.

## Steps
1. Review the analysis data; it enumerates the pods and related messages regarding which images had this issue.
2. The detailed messages might provide specific instructions for the registry that is involved. For example, it might provide a link to instructions on how to increase the limit.

## Related information
* [Increase Rate Limits](https://www.docker.com/increase-rate-limit)
