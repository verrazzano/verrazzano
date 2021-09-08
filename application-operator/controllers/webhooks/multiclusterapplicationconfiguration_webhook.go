// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"net/http"

	"github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	k8sadmission "k8s.io/api/admission/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// MultiClusterApplicationConfigurationValidator is a struct holding objects used during validation.
type MultiClusterApplicationConfigurationValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

// InjectClient injects the client.
func (v *MultiClusterApplicationConfigurationValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *MultiClusterApplicationConfigurationValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle performs validation of created or updated MultiClusterApplicationConfiguration resources.
func (v *MultiClusterApplicationConfigurationValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	mcac := &v1alpha1.MultiClusterApplicationConfiguration{}
	err := v.decoder.Decode(req, mcac)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if mcac.ObjectMeta.DeletionTimestamp.IsZero() {
		switch req.Operation {
		case k8sadmission.Create, k8sadmission.Update:
			err = validateMultiClusterResource(v.client, mcac)
			if err != nil {
				return admission.Denied(err.Error())
			}
			err = validateNamespaceInProject(v.client, mcac.Namespace)
			if err != nil {
				return admission.Denied(err.Error())
			}
		}
	}
	return admission.Allowed("")
}

// Validate that the namespace of the given multiclusterapplicationconfiguration resource is part
// of a verrazzanoproject
func validateNamespaceInProject(c client.Client, namespace string) error {
	vzProjects := v1alpha1.VerrazzanoProjectList{}
	err := c.List(context.TODO(), &vzProjects)
	if err != nil {
		return err
	}

	/*	vzProjects := unstructured.UnstructuredList{}
		vzProjects.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "clusters.verrazzano.io",
			Version: "v1alpha1",
			Kind:    "VerrazzanoProjectList",
		})

		// Get a list of verrazzanoproject resources
		err = c.List(context.TODO(), &vzProjects)
		if err != nil {
			return err
		}
	*/

	if len(vzProjects.Items) == 0 {
		return fmt.Errorf("Namespace %s not specified in any verrazzanoproject resources.  No verrazzanoproject resources found.", namespace)
	}

	// Check verrazzanoProjects for a matching namespace
	for _, proj := range vzProjects.Items {
		for _, ns := range proj.Spec.Template.Namespaces {
			if ns.Metadata.Name == namespace {
				return nil
			}
		}
	}

	/*
		// Check verrazzanoProjects for a matching namespace
		for _, proj := range vzProjects.Items {
			namespaces, _, err := unstructured.NestedSlice(proj.Object, "spec", "template", "namespaces")
			if err != nil {
				return err
			}
			for _, ns := range namespaces {
				u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&ns)
				if err != nil {
					return err
				}
				name, _, err := unstructured.NestedString(u, "metadata", "name")
				if err != nil {
					return err
				}
				if name == namespace {
					return nil
				}
			}
		}
	*/
	// No matching namespace found so return error
	return fmt.Errorf("Namespace %s not specified in any verrazzanoproject resources", namespace)
}
