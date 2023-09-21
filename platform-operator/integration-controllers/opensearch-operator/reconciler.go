// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"time"

	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Reconcile reconciles the IntegrateSingleRequestEvent (in the form of a configmap)
// to perform integration for a single module. Certain modules, such as prometheus-operator,
// require that all integration charts for other modules be installed/upgraded. So in addition
// to applying the chart for a single module, this reconciler may create second event,
// the IntegrateCascadeRequestEvent which processed by the cascade integration controller.
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
		ControllerName: "opensearch",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for opensearch controller: %v", err)
	}
	r.log = log

	// Get effective CR.  Both actualCR and effectiveCR are needed for reconciling
	// Always use actualCR when updating status
	effectiveCR, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	/*********************
	* Add default index patterns
	**********************/

	if *effectiveCR.Spec.Components.Kibana.Enabled && *effectiveCR.Spec.Components.Elasticsearch.Enabled {
		if !effectiveCR.Spec.Components.Elasticsearch.DisableDefaultPolicy {
			zap.S().Infof("DisableDefaultPolicy false")
			err = r.CreateDefaultISMPolicies(controllerCtx, effectiveCR)
			if err != nil {
				return result.NewResultShortRequeueDelayWithError(err)
			}
		} else {
			zap.S().Infof("DisableDefaultPolicy true")
			err = r.DeleteDefaultISMPolicies(controllerCtx, effectiveCR)
			if err != nil {
				return result.NewResultShortRequeueDelayWithError(err)
			}
		}
		if len(effectiveCR.Spec.Components.Elasticsearch.Policies) > 0 {
			err = r.ConfigureISMPolicies(controllerCtx, effectiveCR)
			if err != nil {
				zap.S().Infof("ConfigureISMPolicies true")
				return result.NewResultShortRequeueDelayWithError(err)
			}
		}
		return result.NewResultRequeueDelay(5, 6, time.Minute)
	}
	return result.NewResult()
}

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

func (r *Reconciler) CreateDefaultISMPolicies(controllerCtx controllerspi.ReconcileContext, vz *vzv1alpha1.Verrazzano) error {
	pas, err := GetVerrazzanoPassword(r.Client)
	if err != nil {
		return err
	}
	osClient := NewOSClient(pas)
	err = osClient.SyncDefaultISMPolicy(r.log, r.Client, vz)
	return err
}

func (r *Reconciler) DeleteDefaultISMPolicies(controllerCtx controllerspi.ReconcileContext, vz *vzv1alpha1.Verrazzano) error {
	pas, err := GetVerrazzanoPassword(r.Client)
	if err != nil {
		return err
	}
	osClient := NewOSClient(pas)
	err = osClient.DeleteDefaultISMPolicy(r.log, r.Client, vz)
	return err
}

func (r *Reconciler) ConfigureISMPolicies(controllerCtx controllerspi.ReconcileContext, vz *vzv1alpha1.Verrazzano) error {
	pas, err := GetVerrazzanoPassword(r.Client)
	if err != nil {
		return err
	}
	osClient := NewOSClient(pas)
	err = osClient.ConfigureISM(r.log, r.Client, vz)
	return err
}
