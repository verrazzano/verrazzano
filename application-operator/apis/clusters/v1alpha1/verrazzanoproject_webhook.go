// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"net/http"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	k8sadmission "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var log = logf.Log.WithName("verrazzanoproject-resource")

// VerrazzanoProjectValidator is a struct holding objects used during VerrazzanoProject validation.
type VerrazzanoProjectValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

// InjectClient injects the client.
func (v *VerrazzanoProjectValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *VerrazzanoProjectValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle performs validation of created or updated VerrazzanoProject resources.
func (v *VerrazzanoProjectValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	p := &VerrazzanoProject{}
	err := v.decoder.Decode(req, p)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	log.Info("client=%v", v.client)
	log.Info("operation=%v", req.Operation)
	log.Info("object=%v", p)

	switch req.Operation {
	case k8sadmission.Create:
		return translateErrorToResponse(validateVerrazzanoProject(v.client, p))
	case k8sadmission.Update:
		return translateErrorToResponse(validateVerrazzanoProject(v.client, p))
	default:
		return admission.Allowed("")
	}
}

// validateVerrazzanoProject performs validation checks on the resource
func validateVerrazzanoProject(c client.Client, vp *VerrazzanoProject)  error {
	if vp.ObjectMeta.Namespace != constants.VerrazzanoMultiClusterNamespace {
		return fmt.Errorf("Namespace for the resource must be %q", constants.VerrazzanoMultiClusterNamespace)
	}

	if len(vp.Spec.Template.Namespaces) == 0 {
		return fmt.Errorf("One or more namespaces must be provided")
	}

	if len(vp.Spec.Placement.Clusters) == 0 {
		return fmt.Errorf("One or more target clusters must be provided")
	}

	if err := validateNetworkPolicies(vp); err != nil {
		return err
	}

	if err := vp.validateNamespaceCanBeUsed(); err != nil {
		return err
	}

	if isLocalClusterAdminCluster(c) {
		if err := validateTargetClustersExist(c, vp); err != nil {
			return err
		}
	}

	return nil
}

// validateNetworkPolicies validates the network polices specified in the project
func validateNetworkPolicies(vp *VerrazzanoProject)  error {
	// Build the set of project namespaces for validation
	nsSet := make(map[string]bool)
	for _, ns := range vp.Spec.Template.Namespaces {
		nsSet[ns.Metadata.Name] = true
	}
	// Validate that the policy applies to a namespace in the project
	for _, policyTemplate := range vp.Spec.Template.NetworkPolicies {
		if ok := nsSet[policyTemplate.Metadata.Namespace]; !ok {
			return fmt.Errorf("namespace %s used in NetworkPolicy %s does not exist in project",
				policyTemplate.Metadata.Namespace, policyTemplate.Metadata.Name)
		}
	}
	return nil
}

func (vp *VerrazzanoProject) validateNamespaceCanBeUsed() error {

	c, err := getControllerRuntimeClient()
	if err != nil {
		return fmt.Errorf("failed to get a runtime client: %s", err)
	}

	projectsList := &VerrazzanoProjectList{}
	listOptions := &client.ListOptions{Namespace: constants.VerrazzanoMultiClusterNamespace}
	err = c.List(context.TODO(), projectsList, listOptions)
	if err != nil {
		return fmt.Errorf("failed to get existing Verrazzano projects: %s", err)
	}

	for _, currentNS := range vp.Spec.Template.Namespaces {
		for _, existingProject := range projectsList.Items {
			if existingProject.Name == vp.Name {
				continue
			}
			for _, existingNS := range existingProject.Spec.Template.Namespaces {
				if existingNS.Metadata.Name == currentNS.Metadata.Name {
					return fmt.Errorf("project namespace %s already being used by project %s. projects cannot share a namespace", existingNS.Metadata.Name, existingProject.Name)
				}
			}
		}
	}
	return nil
}

// validateTargetClustersExist determines if all of the target clusters of the project have
// corresponding managed cluster resources.
func validateTargetClustersExist(c client.Client, p *VerrazzanoProject) error {
	for _, cluster := range p.Spec.Placement.Clusters {
		key := client.ObjectKey{Name: cluster.Name, Namespace: constants.VerrazzanoMultiClusterNamespace}
		vmc := v1alpha1.VerrazzanoManagedCluster{}
		err := c.Get(context.TODO(), key, &vmc)
		if err != nil {
			return err
		}
	}
	return nil
}

// isLocalClusterAdminCluster determines if the local cluster is the admin cluster.
func isLocalClusterAdminCluster(c client.Client) bool {
	s := v1.Secret{}
	k := client.ObjectKey{Name: "verrazzano-cluster-registration", Namespace: constants.VerrazzanoSystemNamespace}
	err := c.Get(context.TODO(), k, &s)
	if err != nil && errors.IsNotFound(err) {
		return true
	}
	return false
}

// translateErrorToResponse translates an error to an admission.Response
func translateErrorToResponse(err error) admission.Response {
	if err == nil {
		return admission.Allowed("")
	}
	return admission.Denied(err.Error())
}
