# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: platform-manifests
platform-manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=$(CRD_PATH)
	# Add copyright headers to the kubebuilder generated CRDs
	./hack/add-crd-header.sh
	./hack/update-codegen.sh "verrazzano:v1alpha1" "verrazzano" "boilerplate.go.txt"
	./hack/update-codegen.sh "clusters:v1alpha1" "clusters" "boilerplate-clusters.go.txt"

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: application-manifests
application-manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role paths="./apis/clusters/..." output:crd:artifacts:config=$(CRD_PATH)
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role paths="./apis/oam/..." output:crd:artifacts:config=$(CRD_PATH)
	# Add copyright headers to the kubebuilder generated CRDs
	./hack/add-crd-header.sh
	# Add copyright headers to the kubebuilder generated manifests
	./hack/add-yml-header.sh PROJECT
	./hack/add-yml-header.sh config/rbac/role.yaml

# Generate code
.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# find or download controller-gen
# download controller-gen if necessary
.PHONY: controller-gen
controller-gen:
ifeq (, $(shell command -v controller-gen))
	$(GO) get sigs.k8s.io/controller-tools/cmd/controller-gen
	$(eval CONTROLLER_GEN=$(GOBIN)/controller-gen)
else
	$(eval CONTROLLER_GEN=$(shell command -v controller-gen))
endif
	@{ \
	set -eu; \
	ACTUAL_CONTROLLER_GEN_VERSION=$$(${CONTROLLER_GEN} --version | awk '{print $$2}') ; \
	if [ "$${ACTUAL_CONTROLLER_GEN_VERSION}" != "${CONTROLLER_GEN_VERSION}" ] ; then \
		echo  "Bad controller-gen version $${ACTUAL_CONTROLLER_GEN_VERSION}, please install ${CONTROLLER_GEN_VERSION}" ; \
	fi ; \
	}
