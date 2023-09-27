// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearchdashboards"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/types"
	"time"

	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Reconcile reconciles the OpenSearch integration configmap.
// The configmap existence is just used as a mechanism to trigger reconcile, there
// is nothing in the configmap that is needed to do reconcile.
func (r Reconciler) Reconcile(controllerCtx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	actualCR, err := r.GetVerrazzanoCR()
	if err != nil {
		vzlog.DefaultLogger().ErrorfThrottled("Failed to get Verrazzano CR for OpenSearch integration operator: %v", err)
		return result.NewResultRequeueDelay(30, 40, time.Second)
	}

	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           actualCR.Name,
		Namespace:      actualCR.Namespace,
		ID:             string(actualCR.UID),
		Generation:     actualCR.Generation,
		ControllerName: "opensearch-integration",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for opensearch-operator controller: %v", err)
	}
	r.log = log

	log.Oncef("Starting OpenSearch integration for index patterns and ISM policies")

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
	if isLegacyOS {
		return result.NewResult()
	}

	// Add default index patterns and ISM policies
	if !areComponentsEnabled(effectiveCR) || !isComponentReady(actualCR, opensearch.ComponentName) ||
		!isComponentReady(actualCR, opensearchdashboards.ComponentName) {
		// both components must be enabled and ready
		return result.NewResultRequeueDelay(1, 2, time.Minute)
	}

	if !effectiveCR.Spec.Components.Elasticsearch.DisableDefaultPolicy {
		zap.S().Infof("creating polciies")
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

// areComponentsEnabled returns true if OpenSearch and OpenSearch Dashboard are both enabled
func areComponentsEnabled(effectiveCR *vzv1alpha1.Verrazzano) bool {
	return vzcr.IsOpenSearchEnabled(effectiveCR) && vzcr.IsOpenSearchDashboardsEnabled(effectiveCR)
}

// isComponentReady returns true if the compoent is ready
func isComponentReady(actualCR *vzv1alpha1.Verrazzano, compName string) bool {
	comp := actualCR.Status.Components[compName]
	return comp != nil && comp.State == vzv1alpha1.CompStateReady
}

// CreateIndexPatterns creates the required index patterns using osd client
func (r Reconciler) CreateIndexPatterns(controllerCtx controllerspi.ReconcileContext, vz *vzv1alpha1.Verrazzano) error {
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
func (r Reconciler) CreateDefaultISMPolicies(controllerCtx controllerspi.ReconcileContext, vz *vzv1alpha1.Verrazzano) error {
	osClient, err := r.getOSClient()
	if err != nil {
		return err
	}
	err = osClient.SyncDefaultISMPolicy(r.log, r.Client, vz)
	return err
}

// DeleteDefaultISMPolicies deletes default ISM polcies from OpenSearch
func (r Reconciler) DeleteDefaultISMPolicies(controllerCtx controllerspi.ReconcileContext, vz *vzv1alpha1.Verrazzano) error {
	osClient, err := r.getOSClient()
	if err != nil {
		return err
	}
	err = osClient.DeleteDefaultISMPolicy(r.log, r.Client, vz)
	return err
}

// ConfigureISMPolicies configures ISM policies added by user in Vz cr
func (r Reconciler) ConfigureISMPolicies(controllerCtx controllerspi.ReconcileContext, vz *vzv1alpha1.Verrazzano) error {
	osClient, err := r.getOSClient()
	if err != nil {
		return err
	}
	err = osClient.ConfigureISM(r.log, r.Client, vz)
	return err
}

// getOSClient gets tbe OS client
func (r Reconciler) getOSClient() (*OSClient, error) {
	pas, err := GetVerrazzanoPassword(r.Client)
	if err != nil {
		return nil, err
	}
	osClient := NewOSClient(pas)
	return osClient, nil
}

func (r Reconciler) GetVerrazzanoCR() (*vzv1alpha1.Verrazzano, error) {
	nsn, err := r.GetVerrazzanoNSN()
	if err != nil {
		return nil, err
	}

	vz := &vzv1alpha1.Verrazzano{}
	if err := r.Client.Get(context.TODO(), *nsn, vz); err != nil {
		return nil, err
	}
	return vz, nil
}

func (r Reconciler) GetVerrazzanoNSN() (*types.NamespacedName, error) {
	vzlist := &vzv1alpha1.VerrazzanoList{}
	if err := r.Client.List(context.TODO(), vzlist); err != nil {
		return nil, err
	}
	if len(vzlist.Items) != 1 {
		return nil, fmt.Errorf("Failed, found %d Verrazzano CRs in the cluster.  There must be exactly 1 Verrazzano CR", len(vzlist.Items))
	}
	vz := vzlist.Items[0]
	return &types.NamespacedName{Namespace: vz.Namespace, Name: vz.Name}, nil
}
