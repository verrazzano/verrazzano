# Third party chart hacks

## [update_prometheus_rules.py](update_prometheus_rules.py)

This script updates Prometheus alerting and recording rules by adding `verrazzano_cluster` to `on` and `by` clauses. `on` and `by` in
Prometheus Rules result in all labels not in the `on` and `by` clauses being dropped from the alerts. Without the `verrazzano_cluster`
label it is impossible to determine which cluster fired the alert (when running more than one cluster).

When upgrading the kube-prometheus-stack and thanos charts from upstream, run this script to update the rules.
