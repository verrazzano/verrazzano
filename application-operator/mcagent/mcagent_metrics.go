// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const prometheusClusterNameLabel = "verrazzano_cluster"

func (s *Syncer) updatePrometheusMonitorsClusterName() error {
	err := s.updateServiceMonitorsClusterName()
	if err != nil {
		return err
	}
	err = s.updatePodMonitorsClusterName()
	return err
}

// updateServiceMonitorsClusterName - updates the cluster name in all ServiceMonitors in the managed
// cluster
func (s *Syncer) updateServiceMonitorsClusterName() error {
	serviceMonitors, err := s.listManagedClusterServiceMonitors()
	if err != nil {
		return err
	}
	for _, svcMonitor := range serviceMonitors.Items {
		_, err = controllerutil.CreateOrUpdate(s.Context, s.LocalClient, svcMonitor, func() error {
			for _, endpoint := range svcMonitor.Spec.Endpoints {
				updated := updateClusterNameRelabelConfigs(endpoint.RelabelConfigs, s.ManagedClusterName)
				if updated {
					s.Log.Infof("Updating managed cluster name to %s for service monitor %s", s.ManagedClusterName, svcMonitor.Name)
				}
			}
			return nil
		})
		if err != nil {
			s.Log.Errorf("Error updating managed cluster name in metrics for service monitor %s: %v", svcMonitor.Name, err)
			return err
		}
	}
	return nil
}

// updatePodMonitorsClusterName - updates the cluster name in all PodMonitors in the managed
// cluster
func (s *Syncer) updatePodMonitorsClusterName() error {
	podMonitors, err := s.listManagedClusterPodMonitors()
	if err != nil {
		return err
	}
	for _, podMonitor := range podMonitors.Items {
		_, err = controllerutil.CreateOrUpdate(s.Context, s.LocalClient, podMonitor, func() error {
			for _, endpoint := range podMonitor.Spec.PodMetricsEndpoints {
				updated := updateClusterNameRelabelConfigs(endpoint.RelabelConfigs, s.ManagedClusterName)
				if updated {
					s.Log.Infof("Updating managed cluster name to %s for pod monitor %s", s.ManagedClusterName, podMonitor.Name)
				}
			}
			return nil
		})
		if err != nil {
			s.Log.Errorf("Error updating managed cluster name in metrics for pod monitor %s: %v", podMonitor.Name, err)
			return err
		}
	}
	return nil
}

// updateClusterNameRelabelConfigs - given a list of relabel configs, finds the one for the
// verrazzano_cluster name target label, and updates the cluster name replacement value
func updateClusterNameRelabelConfigs(configs []*v1.RelabelConfig, clusterName string) bool {
	updated := false
	for _, relabelCfg := range configs {
		if relabelCfg.TargetLabel == prometheusClusterNameLabel {
			if relabelCfg.Replacement != clusterName {
				updated = true
			}
			relabelCfg.Replacement = clusterName
		}
	}
	return updated
}

func (s *Syncer) listManagedClusterServiceMonitors() (*v1.ServiceMonitorList, error) {
	listOptions := &client.ListOptions{Namespace: ""}
	serviceMonitorsList := v1.ServiceMonitorList{}
	err := s.LocalClient.List(s.Context, &serviceMonitorsList, listOptions)
	// Only return an error if the Prometheus CRDs are installed
	if err != nil && !meta.IsNoMatchError(err) {
		return nil, err
	}
	return &serviceMonitorsList, nil
}

func (s *Syncer) listManagedClusterPodMonitors() (*v1.PodMonitorList, error) {
	listOptions := &client.ListOptions{Namespace: ""}
	podMonitorsList := v1.PodMonitorList{}
	err := s.LocalClient.List(s.Context, &podMonitorsList, listOptions)
	// Only return an error if the Prometheus CRDs are installed
	if err != nil && !meta.IsNoMatchError(err) {
		return nil, err
	}
	return &podMonitorsList, nil
}
