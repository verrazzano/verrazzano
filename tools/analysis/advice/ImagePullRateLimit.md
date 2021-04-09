# Summary
Analysis detected that there were pods which had issues due to failures to pull an image or images.

The root cause was rate limit exceeded errors while pulling images.

# Steps
1. Review the analysis data supplied. It enumerate the pods and related messages about which images had this issue.
2. The messages detailed may provide specific instructions for the registry that is involved. For example, it may provide a link to instructions on how to increase the limit.

# Related Information
* https://www.docker.com/increase-rate-limit
