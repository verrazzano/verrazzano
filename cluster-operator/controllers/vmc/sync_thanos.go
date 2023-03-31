// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"fmt"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/thanos"
	"google.golang.org/protobuf/types/known/wrapperspb"
	istionet "istio.io/api/networking/v1beta1"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1beta1"
	v1 "k8s.io/api/core/v1"
	k8sapiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	serviceEntryCRDName    = "serviceentries.networking.istio.io"
	destinationRuleCRDName = "destinationrules.networking.istio.io"
	thanosGrpcIngressPort  = 443
)

// thanosServiceDiscovery represents one element in the Thanos service discovery YAML. The YAML
// format contains a list of thanosServiceDiscovery elements
type thanosServiceDiscovery struct {
	Targets []string `json:"targets"`
}

const ThanosManagedClusterEndpointsConfigMap = "verrazzano-thanos-endpoints"
const serviceDiscoveryKey = "servicediscovery.yml"

// syncThanosQuery will perform the necessary sync to make sure Thanos Query on admin cluster can
// talk to Thanos Query on managed cluster (this involves updating the endpoints ConfigMap and
// the Istio config needed for TLS communication to managed cluster)
// TODO - we will also need to add the cluster's CA cert for Thanos Query to use
func (r *VerrazzanoManagedClusterReconciler) syncThanosQuery(ctx context.Context,
	vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {

	if vmc.Status.ThanosHost == "" {
		r.log.Oncef("Managed cluster Thanos Host not found in VMC Status for VMC %s. Not updating Thanos endpoints", vmc.Name)
		return nil
	}

	if err := r.syncThanosQueryEndpoint(ctx, vmc); err != nil {
		return err
	}

	if err := r.createOrUpdateServiceEntry(vmc.Name, vmc.Status.ThanosHost, thanosGrpcIngressPort); err != nil {
		return err
	}
	if err := r.createOrUpdateDestinationRule(vmc.Name, vmc.Status.ThanosHost, thanosGrpcIngressPort); err != nil {
		return err
	}
	return nil
}

// syncThanosQueryEndpoint will update the config map used by Thanos Query with the managed cluster
// Thanos store API endpoint.
func (r *VerrazzanoManagedClusterReconciler) syncThanosQueryEndpoint(ctx context.Context,
	vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {

	if thanosEnabled, err := r.isThanosEnabled(); err != nil || !thanosEnabled {
		r.log.Oncef("Thanos is not enabled on this cluster. Not updating Thanos endpoints for VMC %s", vmc.Name)
		return nil
	}

	return r.addThanosHostIfNotPresent(ctx, vmc.Status.ThanosHost)
}

func (r *VerrazzanoManagedClusterReconciler) deleteClusterThanosEndpoint(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	if vmc.Status.ThanosHost == "" {
		r.log.Oncef("Managed cluster Thanos Host not found in VMC Status for VMC %s. No Thanos endpoint to be deleted", vmc.Name)
		return nil
	}

	if thanosEnabled, err := r.isThanosEnabled(); err != nil || !thanosEnabled {
		r.log.Oncef("Thanos is not enabled on this cluster. Not updating Thanos endpoints for VMC %s", vmc.Name)
		return nil
	}

	if err := r.removeThanosHostFromConfigMap(ctx, vmc.Status.ThanosHost, r.log); err != nil {
		return err
	}
	if err := r.deleteDestinationRule(vmc.Name); err != nil {
		return err
	}
	if err := r.deleteServiceEntry(vmc.Name); err != nil {
		return err
	}
	return nil
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
			return r.createOrUpdateThanosEndpointConfigMap(ctx, serviceDiscoveryList, hostEndpoint, configMap)
		}
	}
	return nil
}

func (r *VerrazzanoManagedClusterReconciler) createOrUpdateThanosEndpointConfigMap(ctx context.Context, serviceDiscoveryList []*thanosServiceDiscovery, hostEndpoint string, configMap *v1.ConfigMap) error {
	newServiceDiscoveryYaml, err := yaml.Marshal(serviceDiscoveryList)
	if err != nil {
		return r.log.ErrorfNewErr("Failed to serialize Thanos endpoints config map content for host %s: %v", hostEndpoint, err)
	}
	_, err = controllerruntime.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		configMap.Data[serviceDiscoveryKey] = string(newServiceDiscoveryYaml)
		return nil
	})
	if err != nil {
		return r.log.ErrorfNewErr("Failed to update Thanos endpoints config map after removing endpoint %s: %v", hostEndpoint, err)
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
		serviceDiscoveryList = []*thanosServiceDiscovery{
			{Targets: []string{}},
		}
	}
	hostEndpoint := toGrpcTarget(host)

	if len(serviceDiscoveryList) == 0 {
		// empty list found, add a placeholder item so that the loop below will populate it
		serviceDiscoveryList = append(serviceDiscoveryList, &thanosServiceDiscovery{})
	}
	for _, serviceDiscovery := range serviceDiscoveryList {
		if serviceDiscovery.Targets == nil {
			serviceDiscovery.Targets = []string{}
		}
		foundIndex := findHost(serviceDiscovery, hostEndpoint)
		if foundIndex > -1 {
			// already exists, nothing to be added
			r.log.Debugf("Managed cluster endpoint %s is already present in the Thanos endpoints config map", hostEndpoint)
			return nil
		}
	}
	// not found, add this host endpoint and update the config map
	r.log.Debugf("Adding managed cluster endpoint %s to Thanos endpoints config map", hostEndpoint)
	serviceDiscoveryList[0].Targets = append(serviceDiscoveryList[0].Targets, hostEndpoint)
	return r.createOrUpdateThanosEndpointConfigMap(ctx, serviceDiscoveryList, hostEndpoint, configMap)
}

func findHost(serviceDiscovery *thanosServiceDiscovery, host string) int {
	foundIndex := -1
	for i, target := range serviceDiscovery.Targets {
		if target == host {
			foundIndex = i
			break
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
	if err := r.Get(ctx, configMapNsn, &configMap); err != nil {
		return nil, r.log.ErrorfNewErr("failed to fetch the Thanos endpoints ConfigMap %s/%s, %v", configMapNsn.Namespace, configMapNsn.Name, err)
	}
	return &configMap, nil
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
	return fmt.Sprintf("%s:%d", hostname, thanosGrpcIngressPort)
}

// createOrUpdateServiceEntry ensures that an Istio ServiceEntry exists for a managed cluster Thanos endpoint. The ServiceEntry is
// used along with a DestinationRule to initiate TLS to the managed cluster ingress. Skip processing if the ServiceEntry CRD
// does not exist in the cluster.
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateServiceEntry(name, host string, port uint32) error {
	isInstalled, err := r.isCRDInstalled(serviceEntryCRDName)
	if err != nil {
		return r.log.ErrorfNewErr("Unable to determine if CRD %s is installed: %v", serviceEntryCRDName, err)
	}
	if !isInstalled {
		r.log.Debugf("CRD %s does not exist in cluster, skipping creating/updating ServiceEntry", serviceEntryCRDName)
		return nil
	}

	// NOTE: We cannot use controller-runtime CreateOrUpdate here because DeepEqual does not work with protobuf-generated types
	se := &istioclinet.ServiceEntry{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: constants.VerrazzanoMonitoringNamespace}}

	// get the ServiceEntry, if it exists we update it, if it does not exist we create it
	err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: name}, se)
	if client.IgnoreNotFound(err) != nil {
		return r.log.ErrorfNewErr("Unable to get ServiceEntry %s/%s: %v", constants.VerrazzanoMonitoringNamespace, name, err)
	}
	if err == nil {
		// we should do some basic fields checks and only update if there are changes, but for now this will have to do
		populateServiceEntry(se, host, port)
		if err = r.Client.Update(context.TODO(), se); err != nil {
			return r.log.ErrorfNewErr("Unable to update ServiceEntry %s/%s: %v", constants.VerrazzanoMonitoringNamespace, name, err)
		}
		return nil
	}

	populateServiceEntry(se, host, port)
	if err = r.Client.Create(context.TODO(), se); err != nil {
		return r.log.ErrorfNewErr("Unable to create ServiceEntry %s/%s: %v", constants.VerrazzanoMonitoringNamespace, name, err)
	}

	return nil
}

func populateServiceEntry(se *istioclinet.ServiceEntry, host string, port uint32) {
	se.Spec.Hosts = []string{host}
	se.Spec.Ports = []*istionet.Port{
		{
			Name:       "grpc",
			Number:     port,
			TargetPort: port,
			Protocol:   "GRPC",
		},
	}
	se.Spec.Resolution = istionet.ServiceEntry_DNS
}

// createOrUpdateDestinationRule ensures that an Istio DestinationRule exists for a managed cluster Thanos endpoint. The DestinationRule is
// used along with a ServiceEntry to initiate TLS to the managed cluster ingress. Skip processing if the DestinationRule CRD
// does not exist in the cluster.
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateDestinationRule(name, host string, port uint32) error {
	isInstalled, err := r.isCRDInstalled(destinationRuleCRDName)
	if err != nil {
		r.log.Errorf("Unable to determine if CRD %s is installed: %v", destinationRuleCRDName, err)
		return err
	}
	if !isInstalled {
		r.log.Debugf("CRD %s does not exist in cluster, skipping creating/updating DestinationRule", destinationRuleCRDName)
		return nil
	}

	// NOTE: We cannot use controller-runtime CreateOrUpdate here because DeepEqual does not work with protobuf-generated types
	dr := &istioclinet.DestinationRule{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: constants.VerrazzanoMonitoringNamespace}}

	// get the DestinationRule, if it exists we update it, if it does not exist we create it
	err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: name}, dr)
	if client.IgnoreNotFound(err) != nil {
		return r.log.ErrorfNewErr("Unable to get DestinationRule %s/%s: %v", constants.VerrazzanoMonitoringNamespace, name, err)
	}
	if err == nil {
		// we should do some basic fields checks and only update if there are changes, but for now this will have to do
		populateDestinationRule(dr, host, port)
		if err = r.Client.Update(context.TODO(), dr); err != nil {
			return r.log.ErrorfNewErr("Unable to update DestinationRule %s/%s: %v", constants.VerrazzanoMonitoringNamespace, name, err)
		}
		return nil
	}

	populateDestinationRule(dr, host, port)
	if err = r.Client.Create(context.TODO(), dr); err != nil {
		return r.log.ErrorfNewErr("Unable to create DestinationRule %s/%s: %v", constants.VerrazzanoMonitoringNamespace, name, err)
	}

	return nil
}

func populateDestinationRule(dr *istioclinet.DestinationRule, host string, port uint32) {
	dr.Spec.Host = host
	dr.Spec.TrafficPolicy = &istionet.TrafficPolicy{
		PortLevelSettings: []*istionet.TrafficPolicy_PortTrafficPolicy{
			{
				Port: &istionet.PortSelector{
					Number: port,
				},
				Tls: &istionet.ClientTLSSettings{
					Mode:               istionet.ClientTLSSettings_SIMPLE,
					InsecureSkipVerify: wrapperspb.Bool(true), // this will be replaced with false when we support managed cluster CA cert checking
				},
			},
		},
	}
}

// isCRDInstalled returns true if the named CRD exists in the cluster, otherwise false.
func (r *VerrazzanoManagedClusterReconciler) isCRDInstalled(crdName string) (bool, error) {
	crd := &k8sapiext.CustomResourceDefinition{}
	err := r.Client.Get(context.TODO(), client.ObjectKey{Name: crdName}, crd)
	if client.IgnoreNotFound(err) != nil {
		return false, err
	}
	return err == nil, nil
}

// deleteServiceEntry deletes an Istio ServiceEntry. No error is returned if the ServiceEntry is not found.
func (r *VerrazzanoManagedClusterReconciler) deleteServiceEntry(name string) error {
	isInstalled, err := r.isCRDInstalled(serviceEntryCRDName)
	if err != nil {
		r.log.Errorf("Unable to determine if CRD %s is installed: %v", serviceEntryCRDName, err)
		return err
	}
	if !isInstalled {
		r.log.Debugf("CRD %s does not exist in cluster, skipping creating/updating DestinationRule", serviceEntryCRDName)
		return nil
	}

	se := &istioclinet.ServiceEntry{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: constants.VerrazzanoMonitoringNamespace}}
	err = r.Client.Delete(context.TODO(), se)
	return client.IgnoreNotFound(err)
}

// deleteDestinationRule deletes an Istio DestinationRule. No error is returned if the DestinationRule is not found.
func (r *VerrazzanoManagedClusterReconciler) deleteDestinationRule(name string) error {
	isInstalled, err := r.isCRDInstalled(destinationRuleCRDName)
	if err != nil {
		r.log.Errorf("Unable to determine if CRD %s is installed: %v", destinationRuleCRDName, err)
		return err
	}
	if !isInstalled {
		r.log.Debugf("CRD %s does not exist in cluster, skipping creating/updating DestinationRule", destinationRuleCRDName)
		return nil
	}

	dr := &istioclinet.DestinationRule{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: constants.VerrazzanoMonitoringNamespace}}
	err = r.Client.Delete(context.TODO(), dr)
	return client.IgnoreNotFound(err)
}
