// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"fmt"
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

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (vp *VerrazzanoProject) ValidateCreate() error {
	log.Info("validate create", "name", vp.Name)

	return vp.validateVerrazzanoProject()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (vp *VerrazzanoProject) ValidateUpdate(old runtime.Object) error {
	log.Info("validate update", "name", vp.Name)

	return vp.validateVerrazzanoProject()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (vp *VerrazzanoProject) ValidateDelete() error {
	log.Info("validate delete", "name", vp.Name)

	// Webhook is not configured for deletes so function will not be called.
	return nil
}

// Perform validation checks on the resource
func (vp *VerrazzanoProject) validateVerrazzanoProject() error {
	if vp.ObjectMeta.Namespace != constants.VerrazzanoMultiClusterNamespace {
		return fmt.Errorf("Namespace for the resource must be %q", constants.VerrazzanoMultiClusterNamespace)
	}

	if len(vp.Spec.Template.Namespaces) == 0 {
		return fmt.Errorf("One or more namespaces must be provided")
	}

	if len(vp.Spec.Placement.Clusters) == 0 {
		return fmt.Errorf("One or more target clusters must be provided")
	}

	if err := vp.validateNetworkPolicies(); err != nil {
		return err
	}

	if err := vp.validateNamespaceCanBeUsed(); err != nil {
		return err
	}

	return nil
}

// Validate the network polices specified in the project
func (vp *VerrazzanoProject) validateNetworkPolicies() error {
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
		return translateErrorToResponse(v.validateCreate(p))
	case k8sadmission.Update:
		return translateErrorToResponse(v.validateUpdate(p))
	default:
		return admission.Allowed("")
	}
}

func translateErrorToResponse(err error) admission.Response {
	if err == nil {
		return admission.Allowed("")
	}
	return admission.Denied(err.Error())
}

func (v *VerrazzanoProjectValidator) validateCreate(p *VerrazzanoProject) error {
	return p.ValidateCreate()
}

func (v *VerrazzanoProjectValidator) validateUpdate(p *VerrazzanoProject) error {
	return p.ValidateUpdate(p)
}
