# Summary
Analysis detected that there were pods which had issues due to failures to pull an image/images.
 
The root cause was rate limit exceeded errors while pulling images.

# Steps
* Review the analysis data supplied. This will enumerate the pods and related messages which provide the images which had this issue.
* The messages detailed may provide specific instructions for the registry that is involved. For example, it may provide a link to instructions on how to increase the limit.

# Related Information
* https://www.docker.com/increase-rate-limit
