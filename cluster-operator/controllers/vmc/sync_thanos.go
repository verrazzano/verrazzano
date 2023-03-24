// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/thanos"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

type thanosServiceDiscovery struct {
	targets []string
}

const ThanosManagedClusterEndpointsConfigMap = "verrazzano-thanos-endpoints"
const serviceDiscoveryKey = "servicediscovery.yml"

// syncThanosQueryEndpoint will update the config map used by Thanos Query with the managed cluster
// Thanos store API endpoint. TODO - we will also need to add the cluster's CA cert for Thanos Query to use
func (r *VerrazzanoManagedClusterReconciler) syncThanosQueryEndpoint(ctx context.Context,
	vmc *clustersv1alpha1.VerrazzanoManagedCluster, log vzlog.VerrazzanoLogger) error {

	if vmc.Status.ThanosHost == "" {
		log.Progressf("Managed cluster Thanos Host not found in VMC Status for VMC %s. Not updating Thanos endpoints", vmc.Name)
		return r.removeThanosHostFromConfigMap(ctx, vmc.Status.ThanosHost, log)
	}

	return r.addThanosHostIfNotPresent(ctx, vmc.Status.ThanosHost, log)
}

func (r *VerrazzanoManagedClusterReconciler) removeThanosHostFromConfigMap(ctx context.Context, host string, log vzlog.VerrazzanoLogger) error {
	configMap, err := r.getThanosEndpointsConfigMap(ctx, log)
	if err != nil {
		return err
	}
	serviceDiscovery, err := parseThanosEndpointsConfigMap(configMap, log)
	if err != nil || serviceDiscovery.targets == nil {
		// nothing to do - there are no endpoints needing removal
		return nil
	}
	hostEndpoint := toGrpcTarget(host)
	foundIndex := findHost(serviceDiscovery, hostEndpoint)
	if foundIndex > -1 {
		serviceDiscovery.targets = append(serviceDiscovery.targets[:foundIndex], serviceDiscovery.targets[foundIndex+1:]...)
		newServiceDiscoveryYaml, err := yaml.Marshal(serviceDiscovery)
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
	return nil
}

func (r *VerrazzanoManagedClusterReconciler) addThanosHostIfNotPresent(ctx context.Context, host string, log vzlog.VerrazzanoLogger) error {
	configMap, err := r.getThanosEndpointsConfigMap(ctx, log)
	if err != nil {
		return err
	}
	serviceDiscovery, err := parseThanosEndpointsConfigMap(configMap, log)
	if err != nil {
		// We will wipe out and repopulate the config map if it could not be parsed
		log.Info("Clearing and repopulating Thanos endpoints ConfigMap due to parse error")
		serviceDiscovery = &thanosServiceDiscovery{targets: []string{}}
	}
	if serviceDiscovery.targets == nil {
		serviceDiscovery.targets = []string{}
	}
	hostEndpoint := toGrpcTarget(host)
	foundIndex := findHost(serviceDiscovery, hostEndpoint)
	if foundIndex > -1 {
		// already exists, nothing to be added
		return nil
	}
	// not found, add this host endpoint and update the config map
	serviceDiscovery.targets = append(serviceDiscovery.targets, hostEndpoint)
	newServiceDiscoveryYaml, err := yaml.Marshal(serviceDiscovery)
	if err != nil {
		return log.ErrorfNewErr("Failed to serialize after adding endpoint %s to Thanos endpoints config map: %v", hostEndpoint, err)
	}
	_, err = controllerruntime.CreateOrUpdate(context.TODO(), r.Client, configMap, func() error {
		configMap.Data[serviceDiscoveryKey] = string(newServiceDiscoveryYaml)
		return nil
	})
	if err != nil {
		return log.ErrorfNewErr("Failed to update Thanos endpoints config map after adding endpoint %s: %v", hostEndpoint, err)
	}
	return nil
}

func findHost(serviceDiscovery *thanosServiceDiscovery, host string) int {
	foundIndex := -1
	for i, target := range serviceDiscovery.targets {
		if target == host {
			foundIndex = i
		}
	}
	return foundIndex
}

func parseThanosEndpointsConfigMap(configMap *v1.ConfigMap, log vzlog.VerrazzanoLogger) (*thanosServiceDiscovery, error) {
	// ConfigMap format for Thanos endpoints is
	// servicediscovery.yml: |
	//  - targets:
	//    - example.com:443
	serviceDiscoveryYaml, exists := configMap.Data[serviceDiscoveryKey]
	serviceDiscoveryContent := thanosServiceDiscovery{}
	var err error
	if exists {
		err = yaml.Unmarshal([]byte(serviceDiscoveryYaml), &serviceDiscoveryContent)
		// TODO if parse fails wipe it out and let it be repopulated
		if err != nil {
			return nil, log.ErrorfNewErr("Failed to parse Thanos endpoints config map %s/%s, error: %v", configMap.Namespace, configMap.Name, err)
		}
	}
	return &serviceDiscoveryContent, nil
}

func (r *VerrazzanoManagedClusterReconciler) getThanosEndpointsConfigMap(ctx context.Context, log vzlog.VerrazzanoLogger) (*v1.ConfigMap, error) {
	configMapNsn := types.NamespacedName{
		Namespace: thanos.ComponentNamespace,
		Name:      ThanosManagedClusterEndpointsConfigMap,
	}
	configMap := v1.ConfigMap{}
	// validate secret if it exists
	if err := r.Get(ctx, configMapNsn, &configMap); err != nil {
		return nil, log.ErrorfNewErr("failed to fetch the Thanos endpoints ConfigMap %s/%s, %v", configMapNsn.Namespace, configMapNsn.Name, err)
	}
	return &configMap, nil
}

// createOrUpdateArgoCDSecret create or update the Argo CD cluster secret
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateThanosConfigMap(configMap *v1.ConfigMap, mutateFn controllerutil.MutateFn) error {
	// Create or update on the local cluster
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), r.Client, configMap, mutateFn)
	return err
}

func toGrpcTarget(hostname string) string {
	return hostname + ":443"
}
