# Summary
Analysis detected that the Verrazzano installation failed while installing the NGINX Ingress Controller.

The root cause appears to be that the load balancer is either missing or unable to set the ingress IP address on the NGINX Ingress service.

## Steps

Refer to the platform-specific environment setup for your platform [here](https://verrazzano.io/docs/setup/platforms/).

## Related information
* [Platform Setup](https://verrazzano.io/docs/setup/platforms/)
* [Kubernetes Troubleshooting](https://kubernetes.io/docs/tasks/debug-application-cluster/troubleshooting/)
