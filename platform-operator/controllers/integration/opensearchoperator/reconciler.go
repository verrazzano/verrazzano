// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"time"

	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Reconcile reconciles the Verrazzano CR
// to perform index pattern creation, ISM Policy creation and configuration
func (r Reconciler) Reconcile(controllerCtx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	// Convert the unstructured to a Verrazzano CR
	actualCR := &vzv1alpha1.Verrazzano{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, actualCR); err != nil {
		controllerCtx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error which should never happen, don't requeue
		return result.NewResult()
	}
	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           actualCR.Name,
		Namespace:      actualCR.Namespace,
		ID:             string(actualCR.UID),
		Generation:     actualCR.Generation,
		ControllerName: "opensearchoperator",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for opensearchoperator controller: %v", err)
	}
	r.log = log

	// Get effective CR.
	effectiveCR, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	componentCtx, err := componentspi.NewContext(log, r.Client, actualCR, nil, false)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	isLegacyOS, err := common.IsLegacyOS(componentCtx)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	// Handle only if its opensearch-operator managed OS
	if !isLegacyOS {
		// Add default index patterns and ISM policies
		if *effectiveCR.Spec.Components.Kibana.Enabled && *effectiveCR.Spec.Components.Elasticsearch.Enabled {
			if !effectiveCR.Spec.Components.Elasticsearch.DisableDefaultPolicy {
				err = r.CreateDefaultISMPolicies(controllerCtx, effectiveCR)
				if err != nil {
					return result.NewResultShortRequeueDelayWithError(err)
				}
			} else {
				err = r.DeleteDefaultISMPolicies(controllerCtx, effectiveCR)
				if err != nil {
					return result.NewResultShortRequeueDelayWithError(err)
				}
			}
			err = r.ConfigureISMPolicies(controllerCtx, effectiveCR)
			if err != nil {
				return result.NewResultShortRequeueDelayWithError(err)
			}
			return result.NewResultRequeueDelay(5, 6, time.Minute)
		}
	}
	return result.NewResult()
}

// CreateIndexPatterns creates the required index patterns using osd client
func (r *Reconciler) CreateIndexPatterns(controllerCtx controllerspi.ReconcileContext, vz *vzv1alpha1.Verrazzano) error {
	pas, err := GetVerrazzanoPassword(r.Client)
	if err != nil {
		return err
	}
	osDashboardsClient := NewOSDashboardsClient(pas)
	osdURL, err := GetOSDHTTPEndpoint(r.Client)
	if err != nil {
		return err
	}
	return osDashboardsClient.CreateDefaultIndexPatterns(r.log, osdURL)
}

// CreateDefaultISMPolicies creates default ISM policies in OpenSearch
func (r *Reconciler) CreateDefaultISMPolicies(controllerCtx controllerspi.ReconcileContext, vz *vzv1alpha1.Verrazzano) error {
	osClient, err := r.getOSClient()
	if err != nil {
		return err
	}
	err = osClient.SyncDefaultISMPolicy(r.log, r.Client, vz)
	return err
}

// DeleteDefaultISMPolicies deletes default ISM polcies from OpenSearch
func (r *Reconciler) DeleteDefaultISMPolicies(controllerCtx controllerspi.ReconcileContext, vz *vzv1alpha1.Verrazzano) error {
	osClient, err := r.getOSClient()
	if err != nil {
		return err
	}
	err = osClient.DeleteDefaultISMPolicy(r.log, r.Client, vz)
	return err
}

// ConfigureISMPolicies configures ISM policies added by user in Vz cr
func (r *Reconciler) ConfigureISMPolicies(controllerCtx controllerspi.ReconcileContext, vz *vzv1alpha1.Verrazzano) error {
	osClient, err := r.getOSClient()
	if err != nil {
		return err
	}
	err = osClient.ConfigureISM(r.log, r.Client, vz)
	return err
}

// getOSClient gets tbe OS client
func (r *Reconciler) getOSClient() (*OSClient, error) {
	pas, err := GetVerrazzanoPassword(r.Client)
	if err != nil {
		return nil, err
	}
	osClient := NewOSClient(pas)
	return osClient, nil
}
