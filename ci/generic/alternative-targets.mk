# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# This is a set of experimental targets

.PHONY: verify-deployment-parallel
verify-deployment-parallel: export TEST_SUITES = opensearch/topology/... examples/helidon/...
verify-deployment-parallel: run-test

.PHONY: verify-deployment-sequential
verify-deployment-sequential: export TEST_SUITES = istio/authz/... metrics/deploymetrics/... logging/system/... logging/opensearch/...  logging/helidon/... examples/helidonmetrics/... workloads/... ingress/console/... loggingtrait/... security/netpol/...
verify-deployment-sequential: run-sequential

.PHONY: verify-all
verify-all: verify-infra-all verify-deployment-all

.PHONY: verify-infra-all
verify-infra-all: verify-install verify-scripts verify-infra verify-security-rbac verify-system-metrics verify-console

.PHONY: verify-install
verify-install:
	${RUNGINKGO} verify-install/...

.PHONY: verify-scripts
verify-scripts:
	${RUNGINKGO} scripts/...

.PHONY: verify-infra
verify-infra:
	${RUNGINKGO} verify-infra/...

.PHONY: verify-security-rbac
verify-security-rbac:
	RUN_PARALLEL=false ${RUNGINKGO} security/rbac/...

.PHONY: verify-system-metrics
verify-system-metrics:
	RUN_PARALLEL=false ${RUNGINKGO} metrics/syscomponents/...

.PHONY: verify-deployment-all
verify-deployment-all: verify-opensearch-topology verify-istio-authz verify-deployment-workload-metrics \
	verify-system-logging verify-opensearch-logging verify-helidon-logging verify-helidon-metrics \
	verify-examples-helidon verify-workloads verify-console-ingress verify-wls-loggingtraits \
	verify-security-netpol

.PHONY: verify-opensearch-topology
verify-opensearch-topology:
	${RUNGINKGO} opensearch/topology/...

.PHONY: verify-istio-authz
verify-istio-authz:
	RUN_PARALLEL=false ${RUNGINKGO} istio/authz/...

.PHONY: verify-deployment-workload-metrics
verify-deployment-workload-metrics:
	RUN_PARALLEL=false ${RUNGINKGO} metrics/deploymetrics/...

.PHONY: verify-system-logging
verify-system-logging:
	RUN_PARALLEL=false ${RUNGINKGO} logging/system/...

.PHONY: verify-opensearch-logging
verify-opensearch-logging:
	RUN_PARALLEL=false ${RUNGINKGO} logging/opensearch/...

.PHONY: verify-helidon-logging
verify-helidon-logging:
	RUN_PARALLEL=false ${RUNGINKGO} logging/helidon/...

.PHONY: verify-helidon-metrics
verify-helidon-metrics:
	RUN_PARALLEL=false ${RUNGINKGO} examples/helidonmetrics/...

.PHONY: verify-examples-helidon
verify-examples-helidon:
	${RUNGINKGO} examples/helidon/...

.PHONY: verify-workloads
verify-workloads:
	RUN_PARALLEL=false ${RUNGINKGO} workloads/...

.PHONY: verify-console-ingress
verify-console-ingress:
	RUN_PARALLEL=false ${RUNGINKGO} ingress/console/...

.PHONY: verify-wls-loggingtraits
verify-wls-loggingtraits:
	RUN_PARALLEL=false ${RUNGINKGO} loggingtrait/...

.PHONY: verify-security-netpol
verify-security-netpol:
	RUN_PARALLEL=false ${RUNGINKGO} security/netpol/...

