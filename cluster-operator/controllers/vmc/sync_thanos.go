// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"encoding/json"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/thanos"
	"google.golang.org/protobuf/types/known/wrapperspb"
	istionet "istio.io/api/networking/v1beta1"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
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
	thanosQueryDeployName  = "thanos-query"
	verrazzanoManagedLabel = "verrazzano_cluster"
	thanosGrpcIngressPort  = 443

	istioVolumeAnnotation      = "sidecar.istio.io/userVolume"
	istioVolumeMountAnnotation = "sidecar.istio.io/userVolumeMount"
	istioVolumeName            = "managed-certs"
	istioCertPath              = "/etc/certs"
)

// thanosServiceDiscovery represents one element in the Thanos service discovery YAML. The YAML
// format contains a list of thanosServiceDiscovery elements
// The format of this object is outlined here https://github.com/thanos-io/thanos/blob/main/docs/service-discovery.md#file-service-discovery
type thanosServiceDiscovery struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

const ThanosManagedClusterEndpointsConfigMap = "verrazzano-thanos-endpoints"
const serviceDiscoveryKey = "servicediscovery.yml"

// syncThanosQuery will perform the necessary sync to make sure Thanos Query on admin cluster can
// talk to Thanos Query on managed cluster (this involves updating the endpoints ConfigMap and
// the Istio config needed for TLS communication to managed cluster)
// TODO - we will also need to add the cluster's CA cert for Thanos Query to use
func (r *VerrazzanoManagedClusterReconciler) syncThanosQuery(ctx context.Context,
	vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {

	if err := r.syncThanosQueryEndpoint(ctx, vmc); err != nil {
		return err
	}
	if err := r.createOrUpdateCACertVolume(vmc, r.addCACertToDeployment); err != nil {
		return err
	}
	if err := r.createOrUpdateServiceEntry(vmc.Name, vmc.Status.ThanosQueryStore, thanosGrpcIngressPort); err != nil {
		return err
	}
	return r.createOrUpdateDestinationRule(vmc, vmc.Status.ThanosQueryStore, thanosGrpcIngressPort)
}

// syncThanosQueryEndpoint will update the config map used by Thanos Query with the managed cluster
// Thanos store API endpoint.
func (r *VerrazzanoManagedClusterReconciler) syncThanosQueryEndpoint(ctx context.Context,
	vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	return r.addThanosHostIfNotPresent(ctx, vmc.Status.ThanosQueryStore, vmc.Name)
}

func (r *VerrazzanoManagedClusterReconciler) syncThanosQueryEndpointDelete(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	if err := r.removeThanosHostFromConfigMap(ctx, vmc.Name, r.log); err != nil {
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

func (r *VerrazzanoManagedClusterReconciler) removeThanosHostFromConfigMap(ctx context.Context, vmcName string, log vzlog.VerrazzanoLogger) error {
	// Check if Thanos is enabled before getting the endpoints ConfigMap
	// to avoid repeating error message when Thanos is disabled
	thanosEnabled, err := r.isThanosEnabled()
	if err != nil || !thanosEnabled {
		return err
	}
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

	for i, serviceDiscovery := range serviceDiscoveryList {
		if findLabelName(serviceDiscovery, vmcName) {
			serviceDiscoveryList = append(serviceDiscoveryList[:i], serviceDiscoveryList[i+1:]...)
			return r.createOrUpdateThanosEndpointConfigMap(ctx, serviceDiscoveryList, vmcName, configMap)
		}
	}
	return nil
}

func (r *VerrazzanoManagedClusterReconciler) createOrUpdateThanosEndpointConfigMap(ctx context.Context, serviceDiscoveryList []*thanosServiceDiscovery, vmcName string, configMap *v1.ConfigMap) error {
	newServiceDiscoveryYaml, err := yaml.Marshal(serviceDiscoveryList)
	if err != nil {
		return r.log.ErrorfNewErr("Failed to serialize Thanos endpoints config map content for VMC %s: %v", vmcName, err)
	}
	result, err := controllerruntime.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		configMap.Data[serviceDiscoveryKey] = string(newServiceDiscoveryYaml)
		return nil
	})
	if err != nil {
		return r.log.ErrorfNewErr("Failed to update Thanos endpoints config map after removing endpoint for VMC %s: %v", vmcName, err)
	}
	if result != controllerutil.OperationResultNone {
		r.log.Infof("The Thanos endpoint Configmap %s has been modified for VMC %s", configMap.Name, vmcName)
	}
	return nil
}

func (r *VerrazzanoManagedClusterReconciler) addThanosHostIfNotPresent(ctx context.Context, host, vmcName string) error {
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

	for i, serviceDiscovery := range serviceDiscoveryList {
		if findLabelAndHost(serviceDiscovery, hostEndpoint, vmcName) {
			// already exists, nothing to be added
			r.log.Debugf("Managed cluster endpoint %s is already present in the Thanos endpoints config map", hostEndpoint)
			return nil
		}
		if findLabelName(serviceDiscovery, vmcName) {
			// label exists, but host has changed
			r.log.Debugf("Modifying managed cluster endpoint %s to Thanos endpoints for VMC %s", hostEndpoint, vmcName)
			serviceDiscoveryList[i] = &thanosServiceDiscovery{
				Targets: []string{hostEndpoint},
				Labels:  serviceDiscovery.Labels,
			}
			return r.createOrUpdateThanosEndpointConfigMap(ctx, serviceDiscoveryList, vmcName, configMap)
		}
	}
	// not found, add this host endpoint and update the config map
	r.log.Debugf("Adding managed cluster endpoint %s to Thanos endpoints config map", hostEndpoint)
	serviceDiscoveryList = append(serviceDiscoveryList, &thanosServiceDiscovery{
		Targets: []string{hostEndpoint},
		Labels: map[string]string{
			verrazzanoManagedLabel: vmcName,
		},
	})
	return r.createOrUpdateThanosEndpointConfigMap(ctx, serviceDiscoveryList, vmcName, configMap)
}

func findLabelAndHost(serviceDiscovery *thanosServiceDiscovery, host, name string) bool {
	if val, ok := serviceDiscovery.Labels[verrazzanoManagedLabel]; !ok || val != name {
		return false
	}
	for _, target := range serviceDiscovery.Targets {
		if target == host {
			return true
		}
	}
	return false
}

// findLabelName parses the service discovery labels and matches it with a given name
func findLabelName(serviceDiscovery *thanosServiceDiscovery, name string) bool {
	val, ok := serviceDiscovery.Labels[verrazzanoManagedLabel]
	return ok && val == name
}

func parseThanosEndpointsConfigMap(configMap *v1.ConfigMap, log vzlog.VerrazzanoLogger) ([]*thanosServiceDiscovery, error) {
	// ConfigMap format for Thanos endpoints is
	// servicediscovery.yml: |
	//  - targets:
	//    - example.com:443
	//    labels:
	//      verrazzano_cluster: managed
	// The format is outlined here: https://github.com/thanos-io/thanos/blob/main/docs/service-discovery.md#file-service-discovery
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
		r.log.Errorf("failed to fetch the Thanos endpoints ConfigMap %s/%s, %v", configMapNsn.Namespace, configMapNsn.Name, err)
		return nil, err
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

// createOrUpdateCACertVolume updates a volume on the Istio Proxy sidecar on the Thanos Deployment to supply CA certs from managed clusters
// This CA cert will be applied to the Destination rule to allow TLS communication to managed cluster query endpoints
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateCACertVolume(vmc *clustersv1alpha1.VerrazzanoManagedCluster, mutation func(*appsv1.Deployment, *clustersv1alpha1.VerrazzanoManagedCluster) error) error {
	queryDeploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoMonitoringNamespace,
			Name:      thanosQueryDeployName,
		},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), r.Client, &queryDeploy, func() error {
		return mutation(&queryDeploy, vmc)
	})
	return err
}

func (r *VerrazzanoManagedClusterReconciler) addCACertToDeployment(queryDeploy *appsv1.Deployment, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	if queryDeploy.Spec.Template.ObjectMeta.Annotations == nil {
		queryDeploy.Spec.Template.ObjectMeta.Annotations = map[string]string{}
	}
	istioVolume, err := r.unmarshalIstioVolumesAnnotation(*queryDeploy)
	if err != nil {
		// Clear the Volume annotation because it has been corrupted
		delete(queryDeploy.Spec.Template.ObjectMeta.Annotations, istioVolumeAnnotation)
		return nil
	}

	if err := r.createCAVolume(queryDeploy, istioVolume, vmc); err != nil {
		return err
	}
	return r.createCAVolumeMount(queryDeploy)
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
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateDestinationRule(vmc *clustersv1alpha1.VerrazzanoManagedCluster, host string, port uint32) error {
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
	dr := &istioclinet.DestinationRule{ObjectMeta: metav1.ObjectMeta{Name: vmc.Name, Namespace: constants.VerrazzanoMonitoringNamespace}}

	// get the DestinationRule, if it exists we update it, if it does not exist we create it
	err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: vmc.Name}, dr)
	if client.IgnoreNotFound(err) != nil {
		return r.log.ErrorfNewErr("Unable to get DestinationRule %s/%s: %v", constants.VerrazzanoMonitoringNamespace, vmc.Name, err)
	}
	if err == nil {
		// we should do some basic fields checks and only update if there are changes, but for now this will have to do
		populateDestinationRule(dr, host, port, vmc)
		if err = r.Client.Update(context.TODO(), dr); err != nil {
			return r.log.ErrorfNewErr("Unable to update DestinationRule %s/%s: %v", constants.VerrazzanoMonitoringNamespace, vmc.Name, err)
		}
		return nil
	}

	populateDestinationRule(dr, host, port, vmc)
	if err = r.Client.Create(context.TODO(), dr); err != nil {
		return r.log.ErrorfNewErr("Unable to create DestinationRule %s/%s: %v", constants.VerrazzanoMonitoringNamespace, vmc.Name, err)
	}

	return nil
}

func populateDestinationRule(dr *istioclinet.DestinationRule, host string, port uint32, vmc *clustersv1alpha1.VerrazzanoManagedCluster) {
	dr.Spec.Host = host
	dr.Spec.TrafficPolicy = &istionet.TrafficPolicy{
		PortLevelSettings: []*istionet.TrafficPolicy_PortTrafficPolicy{
			{
				Port: &istionet.PortSelector{
					Number: port,
				},
				Tls: &istionet.ClientTLSSettings{
					Mode:               istionet.ClientTLSSettings_SIMPLE,
					InsecureSkipVerify: wrapperspb.Bool(false),
					CaCertificates:     fmt.Sprintf("%s/%s", istioCertPath, getCAKey(vmc)),
					Sni:                host,
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

// createCAVolumeMount Adds the CA Volume mount as an Istio annotation to the deployment if it does not exist
func (r *VerrazzanoManagedClusterReconciler) createCAVolumeMount(deployment *appsv1.Deployment) error {
	if _, ok := deployment.Spec.Template.Annotations[istioVolumeMountAnnotation]; !ok {
		volumeMount := []v1.VolumeMount{
			{
				Name:      istioVolumeName,
				MountPath: istioCertPath,
			},
		}
		volumeMountJSON, err := json.Marshal(volumeMount)
		if err != nil {
			return r.log.ErrorfNewErr("Failed to marshal VolumeMount object for Volume %s Istio Annotation: %v", istioVolumeName, err)
		}
		deployment.Spec.Template.Annotations[istioVolumeMountAnnotation] = string(volumeMountJSON)
	}
	return nil
}

// unmarshalIstioVolumesAnnotations returns the Volumes pod annotations from a deployment object
func (r *VerrazzanoManagedClusterReconciler) unmarshalIstioVolumesAnnotation(deployment appsv1.Deployment) (v1.Volume, error) {
	istioVolume := []v1.Volume{{Name: istioVolumeName}}
	podAnnotations := deployment.Spec.Template.ObjectMeta.Annotations
	if volumeJSON, volumeOk := podAnnotations[istioVolumeAnnotation]; volumeOk {
		err := json.Unmarshal([]byte(volumeJSON), &istioVolume)
		if err != nil {
			return istioVolume[0], r.log.ErrorfNewErr("Failed to unmarshal the Istio volume annotation, clearing the volume annotations due to corruption: %v", err)
		}
	}
	return istioVolume[0], nil
}

// createCAVolume adds the CA mount given a Volume object from the Istio annotations and creates the annotation on the deployment
func (r *VerrazzanoManagedClusterReconciler) createCAVolume(deployment *appsv1.Deployment, volume v1.Volume, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	if volume.Secret == nil {
		volume.Secret = &v1.SecretVolumeSource{
			SecretName: constants.PromManagedClusterCACertsSecretName,
		}
	}
	volumeJSON, err := json.Marshal([]v1.Volume{volume})
	if err != nil {
		return r.log.ErrorfNewErr("Failed to marshal Volume %s Istio Annotation: %v", istioVolumeName, err)
	}
	deployment.Spec.Template.Annotations[istioVolumeAnnotation] = string(volumeJSON)
	return nil
}
