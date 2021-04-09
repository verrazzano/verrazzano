# Summary
Analysis detected that the Verrazzano install failed while installing the NGINX Ingress Controller.

The root cause appears to be an OCI IP non-ephemeral address limit has been reached.

# Steps
1. Review the messages from the supporting details for the exact limit.
2. Refer to the OCI documentation related to managing IP Addresses: https://docs.oracle.com/en-us/iaas/Content/Network/Tasks/managingpublicIPs.htm#overview

# Related Information
* https://docs.oracle.com/en-us/iaas/Content/Network/Tasks/managingpublicIPs.htm#overview
