// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v13 "k8s.io/api/networking/v1"
	v12 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Syncer contains context for synchronize operations
type Syncer struct {
	AdminClient        client.Client
	LocalClient        client.Client
	Log                *zap.SugaredLogger
	ManagedClusterName string
	Context            context.Context

	// List of namespaces to watch for multi-cluster objects.
	ProjectNamespaces   []string
	StatusUpdateChannel chan clusters.StatusUpdateMessage
}

type adminStatusUpdateFuncType = func(name types.NamespacedName, newCond clustersv1alpha1.Condition, newClusterStatus clustersv1alpha1.ClusterLevelStatus) error

const retryCount = 3
const managedClusterLabel = "verrazzano.io/managed-cluster"
const mcAppConfigsLabel = "verrazzano.io/mc-app-configs"

var (
	retryDelay = 3 * time.Second
)

// Check if the placement is for this cluster
func (s *Syncer) isThisCluster(placement clustersv1alpha1.Placement) bool {
	// Loop through the cluster list looking for the cluster name
	for _, cluster := range placement.Clusters {
		if cluster.Name == s.ManagedClusterName {
			return true
		}
	}
	return false
}

// processStatusUpdates monitors the StatusUpdateChannel for any
// received messages and processes a batch of them
func (s *Syncer) processStatusUpdates() {
	for i := 0; i < constants.StatusUpdateBatchSize; i++ {
		// Use a select with default so as to not block on the channel if there are no updates
		select {
		case msg := <-s.StatusUpdateChannel:
			err := s.performAdminStatusUpdate(msg)
			if err != nil {
				s.Log.Errorf("Failed to update status on admin cluster for %s/%s from cluster %s after %d retries: %v",
					msg.Resource.GetNamespace(), msg.Resource.GetName(),
					msg.NewClusterStatus.Name, retryCount, err)
			}
		default:
			break
		}
	}
}

// getVerrazzanoManagedNamespaces - return the list of namespaces that have the Verrazzano managed label set to true
func (s *Syncer) getManagedNamespaces() ([]string, error) {
	nsListSelector, err := labels.Parse(fmt.Sprintf("%s=%s", vzconst.VerrazzanoManagedLabelKey, constants.LabelVerrazzanoManagedDefault))
	if err != nil {
		return nil, fmt.Errorf("failed to create list selector on local cluster: %v", err)
	}
	listOptionsGC := &client.ListOptions{LabelSelector: nsListSelector}

	// Get the list of namespaces that were created or managed by VerrazzanoProjects
	vpNamespaceList := corev1.NamespaceList{}
	err = s.LocalClient.List(s.Context, &vpNamespaceList, listOptionsGC)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of Verrazzano managed namespaces: %v", err)
	}

	// Convert the result to a list of strings
	var nsList []string
	for _, namespace := range vpNamespaceList.Items {
		nsList = append(nsList, namespace.Name)
	}

	return nsList, nil
}

func (s *Syncer) performAdminStatusUpdate(msg clusters.StatusUpdateMessage) error {
	fullResourceName := types.NamespacedName{Name: msg.Resource.GetName(), Namespace: msg.Resource.GetNamespace()}
	typeName := reflect.TypeOf(msg.Resource).String()
	var statusUpdateFunc adminStatusUpdateFuncType
	if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.MultiClusterApplicationConfiguration{}).String()) {
		statusUpdateFunc = s.updateMultiClusterAppConfigStatus
	} else if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.MultiClusterComponent{}).String()) {
		statusUpdateFunc = s.updateMultiClusterComponentStatus
	} else if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.MultiClusterConfigMap{}).String()) {
		statusUpdateFunc = s.updateMultiClusterConfigMapStatus
	} else if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.MultiClusterSecret{}).String()) {
		statusUpdateFunc = s.updateMultiClusterSecretStatus
	} else if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.VerrazzanoProject{}).String()) {
		statusUpdateFunc = s.updateVerrazzanoProjectStatus
	} else {
		return fmt.Errorf("received status update message for unknown resource type %s", typeName)
	}
	return s.adminStatusUpdateWithRetry(statusUpdateFunc, fullResourceName, msg.NewCondition, msg.NewClusterStatus)
}

func (s *Syncer) adminStatusUpdateWithRetry(statusUpdateFunc adminStatusUpdateFuncType,
	name types.NamespacedName,
	condition clustersv1alpha1.Condition,
	clusterStatus clustersv1alpha1.ClusterLevelStatus) error {
	var err error
	for tries := 0; tries < retryCount; tries++ {
		err = statusUpdateFunc(name, condition, clusterStatus)
		if err == nil {
			break
		}
		if !errors.IsConflict(err) {
			break
		}

		time.Sleep(retryDelay)
	}
	return err
}

func (s *Syncer) updateVMCStatus() error {
	vmcName := client.ObjectKey{Name: s.ManagedClusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}
	vmc := v1alpha1.VerrazzanoManagedCluster{}
	err := s.AdminClient.Get(s.Context, vmcName, &vmc)
	if err != nil {
		return err
	}

	curTime := v1.Now()
	vmc.Status.LastAgentConnectTime = &curTime
	apiURL, err := s.getAPIServerURL()
	if err != nil {
		return fmt.Errorf("Failed to get api server url for vmc %s with error %v", vmcName, err)
	}

	vmc.Status.APIUrl = apiURL
	prometheusHost, err := s.getPrometheusHost()
	if err != nil {
		return fmt.Errorf("Failed to get api prometheus host to update VMC %s: %v", vmcName, err)
	}
	if prometheusHost != "" {
		vmc.Status.PrometheusHost = prometheusHost
	}

	// Get the Thanos API ingress URL from the local managed cluster, and populate
	// it in the VMC status on the admin cluster, so that admin cluster's Thanos query wire up
	// to the managed cluster
	thanosAPIHost, err := s.getThanosQueryStoreAPIHost()
	if err != nil {
		return fmt.Errorf("Failed to get Thanos query URL to update VMC %s: %v", vmcName, err)
	}

	// If Thanos is disabled, we want to empty the host so Prometheus federation returns
	vmc.Status.ThanosQueryStore = thanosAPIHost

	// update status of VMC
	return s.AdminClient.Status().Update(s.Context, &vmc)
}

// SyncMultiClusterResources - sync multi-cluster objects
func (s *Syncer) SyncMultiClusterResources() {
	// if the MultiClusterApplicationConfiguration CRD does not exist, the other MC resources are
	// unlikely to exist, and we don't need to sync the resources
	mcAppConfCRD := v12.CustomResourceDefinition{}
	if err := s.LocalClient.Get(s.Context,
		types.NamespacedName{Name: mcAppConfCRDName}, &mcAppConfCRD); err != nil {
		if errors.IsNotFound(err) {
			s.Log.Debugf("CRD %s not found - skip syncing multicluster resources", mcAppConfCRDName)
			return
		}
		s.Log.Errorf("Failed retrieving CRD %s: %v", mcAppConfCRDName, err)
	}
	err := s.syncVerrazzanoProjects()
	if err != nil {
		s.Log.Errorf("Failed syncing VerrazzanoProject objects: %v", err)
	}

	// Synchronize objects one namespace at a time
	for _, namespace := range s.ProjectNamespaces {
		err = s.syncSecretObjects(namespace)
		if err != nil {
			s.Log.Errorf("Failed to sync Secret objects: %v", err)
		}
		err = s.syncMCSecretObjects(namespace)
		if err != nil {
			s.Log.Errorf("Failed to sync MultiClusterSecret objects: %v", err)
		}
		err = s.syncMCConfigMapObjects(namespace)
		if err != nil {
			s.Log.Errorf("Failed to sync MultiClusterConfigMap objects: %v", err)
		}
		err = s.syncMCComponentObjects(namespace)
		if err != nil {
			s.Log.Errorf("Failed to sync MultiClusterComponent objects: %v", err)
		}
		err = s.syncMCApplicationConfigurationObjects(namespace)
		if err != nil {
			s.Log.Errorf("Failed to sync MultiClusterApplicationConfiguration objects: %v", err)
		}

		s.processStatusUpdates()

	}
}

// getAPIServerURL returns the API Server URL for Verrazzano instance.
func (s *Syncer) getAPIServerURL() (string, error) {
	ingress := &v13.Ingress{}
	err := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: constants.VzConsoleIngress, Namespace: constants.VerrazzanoSystemNamespace}, ingress)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("Unable to fetch ingress %s/%s, %v", constants.VerrazzanoSystemNamespace, constants.VzConsoleIngress, err)
	}
	return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host), nil
}

// getPrometheusHost returns the prometheus host for Verrazzano instance.
func (s *Syncer) getPrometheusHost() (string, error) {
	ingress := &v13.Ingress{}
	err := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: constants.VzPrometheusIngress, Namespace: constants.VerrazzanoSystemNamespace}, ingress)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("unable to fetch ingress %s/%s, %v", constants.VerrazzanoSystemNamespace, constants.VzPrometheusIngress, err)
	}
	return ingress.Spec.Rules[0].Host, nil
}

// getThanosQueryStoreAPIHost returns the Thanos Query Store API Endpoint URL for Verrazzano instance.
func (s *Syncer) getThanosQueryStoreAPIHost() (string, error) {
	ingress := &v13.Ingress{}
	err := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: vzconst.ThanosQueryStoreIngress, Namespace: constants.VerrazzanoSystemNamespace}, ingress)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("unable to fetch ingress %s/%s, %v", constants.VerrazzanoSystemNamespace, constants.VzPrometheusIngress, err)
	}
	return ingress.Spec.Rules[0].Host, nil
}
