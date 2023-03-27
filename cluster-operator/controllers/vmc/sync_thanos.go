// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/thanos"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

// thanosServiceDiscovery represents one element in the Thanos service discovery YAML. The YAML
// format contains a list of thanosServiceDiscovery elements
type thanosServiceDiscovery struct {
	Targets []string `json:"targets"`
}

const ThanosManagedClusterEndpointsConfigMap = "verrazzano-thanos-endpoints"
const serviceDiscoveryKey = "servicediscovery.yml"

// syncThanosQueryEndpoint will update the config map used by Thanos Query with the managed cluster
// Thanos store API endpoint. TODO - we will also need to add the cluster's CA cert for Thanos Query to use
func (r *VerrazzanoManagedClusterReconciler) syncThanosQueryEndpoint(ctx context.Context,
	vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {

	if vmc.Status.ThanosHost == "" {
		r.log.Progressf("Managed cluster Thanos Host not found in VMC Status for VMC %s. Not updating Thanos endpoints", vmc.Name)
		return nil
	}

	if thanosEnabled, err := r.isThanosEnabled(); err != nil || !thanosEnabled {
		r.log.Oncef("Thanos is not enabled on this cluster. Not updating Thanos endpoints", vmc.Name)
	}

	return r.addThanosHostIfNotPresent(ctx, vmc.Status.ThanosHost)
}

func (r *VerrazzanoManagedClusterReconciler) deleteClusterThanosEndpoint(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	if vmc.Status.ThanosHost == "" {
		r.log.Oncef("Managed cluster Thanos Host not found in VMC Status for VMC %s. No Thanos endpoint to be deleted", vmc.Name)
		return nil
	}

	if thanosEnabled, err := r.isThanosEnabled(); err != nil || !thanosEnabled {
		r.log.Oncef("Thanos is not enabled on this cluster. Not updating Thanos endpoints", vmc.Name)
	}

	return r.removeThanosHostFromConfigMap(ctx, vmc.Status.ThanosHost, r.log)
}

func (r *VerrazzanoManagedClusterReconciler) removeThanosHostFromConfigMap(ctx context.Context, host string, log vzlog.VerrazzanoLogger) error {
	configMap, err := r.getThanosEndpointsConfigMap(ctx)
	if err != nil {
		return err
	}
	serviceDiscoveryList, err := parseThanosEndpointsConfigMap(configMap, log)
	if err != nil {
		// nothing to do - we can't remove entries from an invalid config map - next time an add happens,
		// we will try to automatically resolve the issue
		return nil
	}
	hostEndpoint := toGrpcTarget(host)

	for _, serviceDiscovery := range serviceDiscoveryList {
		foundTargetIndex := findHost(serviceDiscovery, hostEndpoint)
		if foundTargetIndex > -1 {
			serviceDiscovery.Targets = append(serviceDiscovery.Targets[:foundTargetIndex], serviceDiscovery.Targets[foundTargetIndex+1:]...)
			newServiceDiscoveryYaml, err := yaml.Marshal(serviceDiscoveryList)
			if err != nil {
				return log.ErrorfNewErr("Failed to serialize after removing endpoint %s from Thanos endpoints config map: %v", hostEndpoint, err)
			}
			_, err = controllerruntime.CreateOrUpdate(context.TODO(), r.Client, configMap, func() error {
				configMap.Data[serviceDiscoveryKey] = string(newServiceDiscoveryYaml)
				return nil
			})
			if err != nil {
				return log.ErrorfNewErr("Failed to update Thanos endpoints config map after removing endpoint %s: %v", hostEndpoint, err)
			}
		}
	}
	return nil
}

func (r *VerrazzanoManagedClusterReconciler) addThanosHostIfNotPresent(ctx context.Context, host string) error {
	configMap, err := r.getThanosEndpointsConfigMap(ctx)
	if err != nil {
		return err
	}
	serviceDiscoveryList, err := parseThanosEndpointsConfigMap(configMap, r.log)
	if err != nil {
		// We will wipe out and repopulate the config map if it could not be parsed
		r.log.Info("Clearing and repopulating Thanos endpoints ConfigMap due to parse error")
		serviceDiscoveryList = []*thanosServiceDiscovery{}
	}
	hostEndpoint := toGrpcTarget(host)
	for _, serviceDiscovery := range serviceDiscoveryList {
		if serviceDiscovery.Targets == nil {
			serviceDiscovery.Targets = []string{}
		}
		foundIndex := findHost(serviceDiscovery, hostEndpoint)
		if foundIndex > -1 {
			// already exists, nothing to be added
			return nil
		}
		// not found, add this host endpoint and update the config map
		serviceDiscovery.Targets = append(serviceDiscovery.Targets, hostEndpoint)

	}
	newServiceDiscoveryYaml, err := yaml.Marshal(serviceDiscoveryList)
	if err != nil {
		return r.log.ErrorfNewErr("Failed to serialize after adding endpoint %s to Thanos endpoints config map: %v", hostEndpoint, err)
	}
	_, err = controllerruntime.CreateOrUpdate(context.TODO(), r.Client, configMap, func() error {
		configMap.Data[serviceDiscoveryKey] = string(newServiceDiscoveryYaml)
		return nil
	})
	if err != nil {
		return r.log.ErrorfNewErr("Failed to update Thanos endpoints config map after adding endpoint %s: %v", hostEndpoint, err)
	}
	return nil
}

func findHost(serviceDiscovery *thanosServiceDiscovery, host string) int {
	foundIndex := -1
	for i, target := range serviceDiscovery.Targets {
		if target == host {
			foundIndex = i
		}
	}
	return foundIndex
}

func parseThanosEndpointsConfigMap(configMap *v1.ConfigMap, log vzlog.VerrazzanoLogger) ([]*thanosServiceDiscovery, error) {
	// ConfigMap format for Thanos endpoints is
	// servicediscovery.yml: |
	//  - Targets:
	//    - example.com:443
	serviceDiscoveryYaml, exists := configMap.Data[serviceDiscoveryKey]
	serviceDiscoveryArray := []*thanosServiceDiscovery{}
	var err error
	if exists {
		err = yaml.Unmarshal([]byte(serviceDiscoveryYaml), &serviceDiscoveryArray)
		// TODO if parse fails wipe it out and let it be repopulated
		if err != nil {
			return nil, log.ErrorfNewErr("Failed to parse Thanos endpoints config map %s/%s, error: %v", configMap.Namespace, configMap.Name, err)
		}
	}
	return serviceDiscoveryArray, nil
}

func (r *VerrazzanoManagedClusterReconciler) getThanosEndpointsConfigMap(ctx context.Context) (*v1.ConfigMap, error) {
	configMapNsn := types.NamespacedName{
		Namespace: thanos.ComponentNamespace,
		Name:      ThanosManagedClusterEndpointsConfigMap,
	}
	configMap := v1.ConfigMap{}
	// validate secret if it exists
	if err := r.Get(ctx, configMapNsn, &configMap); err != nil {
		return nil, r.log.ErrorfNewErr("failed to fetch the Thanos endpoints ConfigMap %s/%s, %v", configMapNsn.Namespace, configMapNsn.Name, err)
	}
	return &configMap, nil
}

// createOrUpdateArgoCDSecret create or update the Argo CD cluster secret
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateThanosConfigMap(configMap *v1.ConfigMap, mutateFn controllerutil.MutateFn) error {
	// Create or update on the local cluster
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), r.Client, configMap, mutateFn)
	return err
}

func (r *VerrazzanoManagedClusterReconciler) isThanosEnabled() (bool, error) {
	vz, err := r.getVerrazzanoResource()
	if err != nil {
		r.log.Errorf("Failed to retrieve Verrazzano CR: %v", err)
		return false, err
	}
	return vzcr.IsThanosEnabled(vz), nil
}

func toGrpcTarget(hostname string) string {
	return hostname + ":443"
}
