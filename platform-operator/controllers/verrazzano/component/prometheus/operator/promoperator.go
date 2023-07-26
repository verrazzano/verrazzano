// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promoperapiv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	securityv1beta1 "istio.io/api/security/v1beta1"
	istiov1beta1 "istio.io/api/type/v1beta1"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	deploymentName  = "prometheus-operator-kube-p-operator"
	istioVolumeName = "istio-certs-dir"
	serviceAccount  = "cluster.local/ns/verrazzano-monitoring/sa/prometheus-operator-kube-p-prometheus"

	prometheusAuthPolicyName = "vmi-system-prometheus-authzpol"
	networkPolicyName        = "vmi-system-prometheus"
	istioCertMountPath       = "/etc/istio-certs"

	prometheusName         = "prometheus"
	alertmanagerName       = "alertmanager"
	alertmanagerConfigName = "alertmanager-config"
	thanosName             = "thanos"
	configReloaderName     = "prometheus-config-reloader"

	pvcName                  = "prometheus-prometheus-operator-kube-p-prometheus-db-prometheus-prometheus-operator-kube-p-prometheus-0"
	defaultPrometheusStorage = "50Gi"

	alertmanagerConfigLabelName    = "verrazzano.io/alerts"
	alertmanagerConfigLabelValue   = "system"
	nullAlertmanagerConfigReceiver = "null-receiver"
)

func prometheusOperatorListOptions() (*client.ListOptions, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			constants.VerrazzanoComponentLabelKey: ComponentName,
		},
	})
	if err != nil {
		return nil, err
	}
	return &client.ListOptions{
		Namespace:     ComponentNamespace,
		LabelSelector: selector,
	}, nil
}

// isPrometheusOperatorReady checks if the Prometheus operator deployment is ready
func isPrometheusOperatorReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	listOptions, err := prometheusOperatorListOptions()
	if err != nil {
		ctx.Log().Errorf("Failed to create selector for %s: %v", ComponentName, err)
		return false
	}
	return ready.DeploymentsReadyBySelectors(ctx.Log(), ctx.Client(), 1, prefix, listOptions)
}

// preInstallUpgrade handles pre-install and pre-upgrade processing for the Prometheus Operator Component
func preInstallUpgrade(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Prometheus Operator preInstallUpgrade dry run")
		return nil
	}

	// Create the verrazzano-monitoring namespace
	ctx.Log().Debugf("Creating/updating namespace %s for the Prometheus Operator", ComponentNamespace)
	ns := common.GetVerrazzanoMonitoringNamespace()
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), ns, func() error {
		common.MutateVerrazzanoMonitoringNamespace(ctx, ns)
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}

	// Create an empty secret for the additional scrape configs - this secret gets populated with scrape jobs for managed clusters
	if err := ensureAdditionalScrapeConfigsSecret(ctx); err != nil {
		return err
	}

	err := common.ApplyCRDYaml(ctx, config.GetHelmPromOpChartsDir())
	if err != nil {
		return err
	}

	// Remove any existing volume claims from old VMO-managed Prometheus persistent volumes
	return updateExistingVolumeClaims(ctx)
}

// postInstallUpgrade handles post-install and post-upgrade processing for the Prometheus Operator Component
func postInstallUpgrade(ctx spi.ComponentContext) error {
	if ctx.IsDryRun() {
		ctx.Log().Debug("Prometheus Operator postInstallUpgrade dry run")
		return nil
	}

	if err := applySystemMonitors(ctx); err != nil {
		return err
	}
	if err := updateApplicationAuthorizationPolicies(ctx); err != nil {
		return err
	}
	if err := reconcileIngresses(ctx); err != nil {
		return err
	}
	if err := createOrUpdatePrometheusAuthPolicy(ctx); err != nil {
		return err
	}
	if err := createOrUpdateServiceMonitors(ctx); err != nil {
		return err
	}
	if err := common.UpdatePrometheusAnnotations(ctx, ComponentNamespace, ComponentName); err != nil {
		return err
	}
	if err := createOrUpdateAlertmanagerConfig(ctx); err != nil {
		return err
	}
	// if there is a persistent volume that was migrated from the VMO-managed Prometheus, make sure the reclaim policy is set
	// back to its original value
	return resetVolumeReclaimPolicy(ctx)
}

// ensureAdditionalScrapeConfigsSecret creates an empty secret for additional scrape configurations loaded by Prometheus, if the secret
// does not already exist. Initially this secret is empty but when managed clusters are created, the federated scrape configuration
// is added to this secret.
func ensureAdditionalScrapeConfigsSecret(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Creating or updating secret %s for Prometheus additional scrape configs", vzconst.PromAdditionalScrapeConfigsSecretName)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vzconst.PromAdditionalScrapeConfigsSecretName,
			Namespace: ComponentNamespace,
		},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		if _, exists := secret.Data[vzconst.PromAdditionalScrapeConfigsSecretKey]; !exists {
			secret.Data[vzconst.PromAdditionalScrapeConfigsSecretKey] = []byte{}
		}
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s secret: %v", vzconst.PromAdditionalScrapeConfigsSecretName, err)
	}
	return nil
}

// updateExistingVolumeClaims removes a persistent volume claim from the Prometheus persistent volume if the
// claim was from the VMO-managed Prometheus and the status is "released". This allows the new Prometheus instance to
// bind to the existing volume.
func updateExistingVolumeClaims(ctx spi.ComponentContext) error {
	ctx.Log().Info("Removing old claim from Prometheus persistent volume if a volume exists")

	pvList, err := getPrometheusPersistentVolumes(ctx)
	if err != nil {
		return err
	}

	// find a volume that has been released but still has a claim for the old VMO-managed Prometheus
	for i := range pvList.Items {
		pv := pvList.Items[i] // avoids "Implicit memory aliasing in for loop" linter complaint
		if pv.Status.Phase != corev1.VolumeReleased {
			continue
		}
		if pv.Spec.ClaimRef != nil && pv.Spec.ClaimRef.Namespace == constants.VerrazzanoSystemNamespace && pv.Spec.ClaimRef.Name == constants.VMISystemPrometheusVolumeClaim {
			ctx.Log().Infof("Found volume, removing old claim from Prometheus persistent volume %s", pv.Name)
			pv.Spec.ClaimRef = nil
			if err := ctx.Client().Update(context.TODO(), &pv); err != nil {
				return ctx.Log().ErrorfNewErr("Failed removing claim from persistent volume %s: %v", pv.Name, err)
			}
			if err := createPVCFromPV(ctx, pv); err != nil {
				return ctx.Log().ErrorfNewErr("Failed to create new PVC from volume %s: %v", pv.Name, err)
			}
			break
		}
	}
	return nil
}

// createPVCFromPV creates a PVC from a PV definition, and sets the PVC to reference the PV by name
func createPVCFromPV(ctx spi.ComponentContext, volume corev1.PersistentVolume) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: ComponentNamespace,
		},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), pvc, func() error {
		accessModes := make([]corev1.PersistentVolumeAccessMode, len(volume.Spec.AccessModes))
		copy(accessModes, volume.Spec.AccessModes)
		pvc.Spec.AccessModes = accessModes
		pvc.Spec.Resources = corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceStorage: volume.Spec.Capacity.Storage().DeepCopy(),
			},
		}
		pvc.Spec.VolumeName = volume.Name
		return nil
	})
	return err
}

// getPrometheusPersistentVolumes returns a volume list containing a Prometheus persistent volume created by
// an older VMO installation
func getPrometheusPersistentVolumes(ctx spi.ComponentContext) (*corev1.PersistentVolumeList, error) {
	pvList := &corev1.PersistentVolumeList{}
	if err := ctx.Client().List(context.TODO(), pvList, client.MatchingLabels{constants.StorageForLabel: constants.PrometheusStorageLabelValue}); err != nil {
		return nil, ctx.Log().ErrorfNewErr("Failed listing persistent volumes: %v", err)
	}
	return pvList, nil
}

// resetVolumeReclaimPolicy resets the reclaim policy on a Prometheus storage volume to its original value. The volume
// would have been created by the VMO for Prometheus and prior to upgrading the VMO, we set the reclaim policy to
// "retain" so that we can migrate it to the new Prometheus. Now that it has been migrated, we reset the reclaim policy
// to its original value.
func resetVolumeReclaimPolicy(ctx spi.ComponentContext) error {
	ctx.Log().Info("Resetting reclaim policy on Prometheus persistent volume if a volume exists")

	pvList, err := getPrometheusPersistentVolumes(ctx)
	if err != nil {
		return err
	}

	for i := range pvList.Items {
		pv := pvList.Items[i] // avoids "Implicit memory aliasing in for loop" linter complaint
		if pv.Status.Phase != corev1.VolumeBound {
			continue
		}
		if pv.Labels == nil {
			continue
		}
		oldPolicy := pv.Labels[constants.OldReclaimPolicyLabel]

		if len(oldPolicy) > 0 {
			// found a bound volume that still has an old reclaim policy label, so reset the reclaim policy and remove the label
			ctx.Log().Infof("Found volume, resetting reclaim policy on Prometheus persistent volume %s to %s", pv.Name, oldPolicy)
			pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimPolicy(oldPolicy)
			delete(pv.Labels, constants.OldReclaimPolicyLabel)

			if err := ctx.Client().Update(context.TODO(), &pv); err != nil {
				return ctx.Log().ErrorfNewErr("Failed resetting reclaim policy on persistent volume %s: %v", pv.Name, err)
			}
			break
		}
	}
	return nil
}

// AppendOverrides appends install overrides for the Prometheus Operator Helm chart
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Append custom images from the subcomponents in the bom
	ctx.Log().Debug("Appending the image overrides for the Prometheus Operator components")
	subcomponents := []string{configReloaderName, alertmanagerName, prometheusName}
	kvs, err := appendCustomImageOverrides(ctx, kvs, subcomponents)
	if err != nil {
		return kvs, err
	}

	// Replace default images for subcomponents Alertmanager and Prometheus
	defaultImages := map[string]string{
		// format "subcomponentName": "helmDefaultKey"
		alertmanagerName: "prometheusOperator.alertmanagerDefaultBaseImage",
		prometheusName:   "prometheusOperator.prometheusDefaultBaseImage",
	}
	kvs, err = appendDefaultImageOverrides(ctx, kvs, defaultImages)
	if err != nil {
		return kvs, err
	}

	kvs, err = appendThanosImageOverrides(ctx, kvs)
	if err != nil {
		return kvs, err
	}

	// If the cert-manager component is enabled, use it for webhook certificates, otherwise Prometheus Operator
	// will use the kube-webhook-certgen image
	kvs = append(kvs, bom.KeyValue{
		Key:   "prometheusOperator.admissionWebhooks.certManager.enabled",
		Value: strconv.FormatBool(vzcr.IsClusterIssuerEnabled(ctx.EffectiveCR())),
	})

	if vzcr.IsPrometheusEnabled(ctx.EffectiveCR()) {
		// If storage overrides are specified, set helm overrides
		resourceRequest, err := common.FindStorageOverride(ctx.EffectiveCR())
		if err != nil {
			return kvs, err
		}
		// If no storage specified (dev specifies emptydir), use 50Gi
		if resourceRequest == nil {
			resourceRequest = &common.ResourceRequestValues{
				Storage: defaultPrometheusStorage,
			}
		}
		if resourceRequest != nil {
			kvs, err = appendResourceRequestOverrides(ctx, resourceRequest, kvs)
			if err != nil {
				return kvs, err
			}
		}

		// Append the Istio Annotations for Prometheus
		kvs, err = appendIstioOverrides("prometheus.prometheusSpec.podMetadata.annotations",
			"prometheus.prometheusSpec.volumeMounts",
			"prometheus.prometheusSpec.volumes",
			kvs)
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed applying the Istio Overrides for Prometheus")
		}

		kvs, err = appendAdditionalVolumeOverrides(ctx,
			"prometheus.prometheusSpec.volumeMounts",
			"prometheus.prometheusSpec.volumes",
			kvs)
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed applying additional volume overrides for Prometheus")
		}
	} else {
		kvs = append(kvs, bom.KeyValue{
			Key:   "prometheus.enabled",
			Value: "false",
		})
	}

	// Add a label to Prometheus Operator resources to distinguish Verrazzano resources
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("commonLabels.%s", constants.VerrazzanoComponentLabelKey), Value: ComponentName})

	// Add label to the Prometheus Operator pod to avoid a sidecar injection
	kvs = append(kvs, bom.KeyValue{Key: `prometheusOperator.podAnnotations.sidecar\.istio\.io/inject`, Value: `"false"`})

	// disable tls if Istio is not enabled
	if !vzcr.IsIstioEnabled(ctx.EffectiveCR()) {
		kvs = append(kvs, bom.KeyValue{Key: "prometheusOperator.tls.enabled", Value: "false"})
	}

	return kvs, nil
}

// appendResourceRequestOverrides adds overrides for persistent storage and memory
func appendResourceRequestOverrides(ctx spi.ComponentContext, resourceRequest *common.ResourceRequestValues, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	storage := resourceRequest.Storage
	memory := resourceRequest.Memory

	if len(storage) > 0 {
		kvs = append(kvs, []bom.KeyValue{
			{
				Key:   "prometheus.prometheusSpec.storageSpec.disableMountSubPath",
				Value: "true",
			},
			{
				Key:   "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage",
				Value: storage,
			},
		}...)
	}
	if len(memory) > 0 {
		kvs = append(kvs, []bom.KeyValue{
			{
				Key:   "prometheus.prometheusSpec.resources.requests.memory",
				Value: memory,
			},
		}...)
	}

	return kvs, nil
}

// appendCustomImageOverrides takes a list of subcomponent image names and appends it to the given Helm overrides
func appendCustomImageOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue, subcomponents []string) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctx.Log().ErrorNewErr("Failed to get the bom file for the Prometheus Operator image overrides: ", err)
	}

	for _, subcomponent := range subcomponents {
		imageOverrides, err := bomFile.BuildImageOverrides(subcomponent)
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed to build the Prometheus Operator image overrides for subcomponent %s: ", subcomponent, err)
		}
		kvs = append(kvs, imageOverrides...)
	}

	return kvs, nil
}

func appendDefaultImageOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue, subcomponents map[string]string) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctx.Log().ErrorNewErr("Failed to get the bom file for the Prometheus Operator image overrides: ", err)
	}

	for subcomponent, helmKey := range subcomponents {
		images, err := bomFile.GetImageNameList(subcomponent)
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed to get the image for subcomponent %s from the bom: ", subcomponent, err)
		}
		if len(images) > 0 {
			// kube-prometheus-stack helm chart now expects the registry to be specified in a separate helm value
			reg, imageWithoutReg, _ := strings.Cut(images[0], "/")
			kvs = append(kvs, bom.KeyValue{Key: helmKey + "Registry", Value: reg})
			kvs = append(kvs, bom.KeyValue{Key: helmKey, Value: imageWithoutReg})
		}
	}

	return kvs, nil
}

// appendThanosImageOverrides appends overrides for the Thanos image in the Prometheus prometheusSpec and
// the default base image in prometheusOperator
func appendThanosImageOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctx.Log().ErrorNewErr("Failed to get the bom file for the Prometheus Operator image overrides: ", err)
	}

	images, err := bomFile.GetImageNameList(thanosName)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to get the image for subcomponent %s from the bom: ", thanosName, err)
	}
	if len(images) > 0 {
		// example: ghcr.io/verrazzano/thanos:v1.2
		// reg = ghcr.io
		// repo = verrazzano/thanos:v1.2
		// repoAndImage = verrazzano/thanos
		// tag = v1.2
		reg, repo, _ := strings.Cut(images[0], "/")
		repoAndImage, tag, _ := strings.Cut(repo, ":")
		kvs = append(kvs, bom.KeyValue{Key: "prometheus.prometheusSpec.thanos.image", Value: images[0]})
		kvs = append(kvs, bom.KeyValue{Key: "prometheusOperator.thanosImage.registry", Value: reg})
		kvs = append(kvs, bom.KeyValue{Key: "prometheusOperator.thanosImage.repository", Value: repoAndImage})
		kvs = append(kvs, bom.KeyValue{Key: "prometheusOperator.thanosImage.tag", Value: tag})
	}

	return kvs, nil
}

// validatePrometheusOperator checks scenarios in which the Verrazzano CR violates install verification due to Prometheus Operator specifications
func (c prometheusComponent) validatePrometheusOperator(vz *installv1beta1.Verrazzano) error {
	// Validate if Prometheus is enabled, Prometheus Operator should be enabled
	if !c.IsEnabled(vz) && vzcr.IsPrometheusEnabled(vz) {
		return fmt.Errorf("Prometheus Operator must be enabled if Prometheus is enabled")
	}
	// Validate install overrides for v1beta1.Verrazzano
	if vz.Spec.Components.PrometheusOperator != nil {
		if err := vzapi.ValidateInstallOverridesV1Beta1(vz.Spec.Components.PrometheusOperator.ValueOverrides); err != nil {
			return err
		}
	}
	return nil
}

// appendIstioOverrides appends Istio annotations necessary for Prometheus in Istio
// Istio is required on the Prometheus for mTLS between it and Verrazzano applications
func appendIstioOverrides(annotationsKey, volumeMountKey, volumeKey string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Istio annotations that will copy the volume mount for the Istio certs to the envoy sidecar
	// The last annotation allows envoy to intercept only requests from the Keycloak Service IP
	annotations := map[string]string{
		`proxy\.istio\.io/config`:                             `{"proxyMetadata":{ "OUTPUT_CERTS": "/etc/istio-output-certs"}}`,
		`sidecar\.istio\.io/userVolumeMount`:                  `[{"name": "istio-certs-dir", "mountPath": "/etc/istio-output-certs"}]`,
		`traffic\.sidecar\.istio\.io/excludeOutboundIPRanges`: "0.0.0.0/0",
	}

	for key, value := range annotations {
		kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s.%s", annotationsKey, key), Value: value})
	}

	// Volume mount on the Prometheus container to mount the Istio-generated certificates
	vm := corev1.VolumeMount{
		Name:      istioVolumeName,
		MountPath: istioCertMountPath,
	}
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[0].name", volumeMountKey), Value: vm.Name})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[0].mountPath", volumeMountKey), Value: vm.MountPath})

	// Volume annotation to enable an in-memory location for Istio to place and serve certificates
	vol := corev1.Volume{
		Name: istioVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	}
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[0].name", volumeKey), Value: vol.Name})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[0].emptyDir.medium", volumeKey), Value: string(vol.VolumeSource.EmptyDir.Medium)})
	return kvs, nil
}

// GetOverrides appends Helm value overrides for the Prometheus Operator Helm chart
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.PrometheusOperator != nil {
			return effectiveCR.Spec.Components.PrometheusOperator.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.PrometheusOperator != nil {
			return effectiveCR.Spec.Components.PrometheusOperator.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

// appendAdditionalVolumeOverrides adds a volume and volume mount so we can mount managed cluster TLS certs from a secret in the Prometheus pod.
// Initially the secret does not exist. When managed clusters are created, the secret is created and Prometheus TLS certs for the managed
// clusters are added to the secret.
func appendAdditionalVolumeOverrides(ctx spi.ComponentContext, volumeMountKey, volumeKey string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[1].name", volumeMountKey), Value: "managed-cluster-ca-certs"})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[1].mountPath", volumeMountKey), Value: "/etc/prometheus/managed-cluster-ca-certs"})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[1].readOnly", volumeMountKey), Value: "true"})

	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[1].name", volumeKey), Value: "managed-cluster-ca-certs"})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[1].secret.secretName", volumeKey), Value: constants.PromManagedClusterCACertsSecretName})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[1].secret.optional", volumeKey), Value: "true"})

	return kvs, nil
}

// applySystemMonitors applies templatized PodMonitor and ServiceMonitor custom resources for Verrazzano system
// components to the cluster
func applySystemMonitors(ctx spi.ComponentContext) error {
	// create template key/value map
	args := make(map[string]interface{})
	args["systemNamespace"] = constants.VerrazzanoSystemNamespace
	args["monitoringNamespace"] = constants.VerrazzanoMonitoringNamespace
	args["nginxNamespace"] = nginxutil.IngressNGINXNamespace()
	args["istioNamespace"] = constants.IstioSystemNamespace
	args["installNamespace"] = constants.VerrazzanoInstallNamespace

	istio := ctx.EffectiveCR().Spec.Components.Istio
	enabled := istio != nil && istio.IsInjectionEnabled()
	args["isIstioEnabled"] = enabled

	// substitute template values to all files in the directory and apply the resulting YAML
	dir := path.Join(config.GetThirdPartyManifestsDir(), "prometheus-operator")
	yamlApplier := k8sutil.NewYAMLApplier(ctx.Client(), "")
	err := yamlApplier.ApplyDT(dir, args)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to substitute template values for System Monitors: %v", err)
	}
	return nil
}

func updateApplicationAuthorizationPolicies(ctx spi.ComponentContext) error {
	// Get the Application namespaces by filtering the label verrazzano-managed=true
	nsList := corev1.NamespaceList{}
	err := ctx.Client().List(context.TODO(), &nsList, &client.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set{vzconst.VerrazzanoManagedLabelKey: "true"})})
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to list namespaces with the label %s=true: %v", vzconst.VerrazzanoManagedLabelKey, err)
	}

	// For each namespace, if an authorization policy exists, add the prometheus operator service account as a principal
	for _, ns := range nsList.Items {
		authPolicyList := istioclisec.AuthorizationPolicyList{}
		err = ctx.Client().List(context.TODO(), &authPolicyList, &client.ListOptions{Namespace: ns.Name})
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed to list Authorization Policies in namespace %s: %v", ns.Name, err)
		}
		// Parse the authorization policy list for the Verrazzano Istio label and apply the service account to the first rule
		for i := range authPolicyList.Items {
			authPolicy := authPolicyList.Items[i]
			if _, ok := authPolicy.Labels[constants.IstioAppLabel]; !ok {
				continue
			}
			_, err = common.CreateOrUpdateProtobuf(context.TODO(), ctx.Client(), authPolicy, func() error {
				rules := authPolicy.Spec.Rules
				if len(rules) <= 0 || rules[0] == nil {
					return nil
				}
				targetRule := rules[0]
				if len(targetRule.From) <= 0 || targetRule.From[0] == nil {
					return nil
				}
				targetFrom := targetRule.From[0]
				if targetFrom.Source == nil {
					return nil
				}
				// Update the object principal with the Prometheus Operator service account if not found
				if !vzstring.SliceContainsString(targetFrom.Source.Principals, serviceAccount) {
					authPolicy.Spec.Rules[0].From[0].Source.Principals = append(targetFrom.Source.Principals, serviceAccount)
				}
				return nil
			})
			if err != nil {
				return ctx.Log().ErrorfNewErr("Failed to update the Authorization Policy %s in namespace %s: %v", authPolicy.Name, ns.Name, err)
			}
		}
	}
	return nil
}

// createOrUpdatePrometheusAuthPolicy creates the Istio authorization policy for Prometheus
func createOrUpdatePrometheusAuthPolicy(ctx spi.ComponentContext) error {
	// if Istio is explicitly disabled, do not attempt to create the auth policy
	istio := ctx.EffectiveCR().Spec.Components.Istio
	if istio != nil && istio.Enabled != nil && !*istio.Enabled {
		return nil
	}

	authPol := istioclisec.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: prometheusAuthPolicyName},
	}
	var rules []*securityv1beta1.Rule

	// allow Auth Proxy, Grafana, and Kiali to access Prometheus
	var serviceAccounts []string
	if vzcr.IsAuthProxyEnabled(ctx.EffectiveCR()) {
		serviceAccounts = append(serviceAccounts, common.GetAuthProxyPrincipal())
	}
	if vzcr.IsVMOEnabled(ctx.EffectiveCR()) {
		serviceAccounts = append(serviceAccounts, common.GetVMOPrincipal())
	}
	if vzcr.IsKialiEnabled(ctx.EffectiveCR()) {
		serviceAccounts = append(serviceAccounts, common.GetKialiPrincipal())
	}
	if len(serviceAccounts) > 0 {
		rules = append(rules, common.ConstructAuthPolicyRule([]string{constants.VerrazzanoSystemNamespace},
			serviceAccounts, []string{"9090", "10901"}))
	}

	// allow Prometheus to scrape its own Envoy sidecar
	rules = append(rules, common.ConstructAuthPolicyRule([]string{ComponentNamespace}, []string{serviceAccount},
		[]string{"9090"}))

	// allow Jaeger to access Prometheus
	if vzcr.IsJaegerOperatorEnabled(ctx.EffectiveCR()) {
		rules = append(rules, common.ConstructAuthPolicyRule([]string{constants.VerrazzanoMonitoringNamespace},
			[]string{common.GetJaegerPrincipal()}, []string{"9090"}))
	}

	// allow Thanos Query to access Prometheus Thanos sidecar on port 10901
	if vzcr.IsThanosEnabled(ctx.EffectiveCR()) {
		rules = append(rules, common.ConstructAuthPolicyRule([]string{constants.VerrazzanoMonitoringNamespace},
			[]string{common.GetThanosQueryPrincipal()}, []string{"10901"}))
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &authPol, func() error {
		authPol.Spec = securityv1beta1.AuthorizationPolicy{
			Selector: &istiov1beta1.WorkloadSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": prometheusName,
				},
			},
			Action: securityv1beta1.AuthorizationPolicy_ALLOW,
			Rules:  rules,
		}
		return nil
	})
	if ctrlerrors.ShouldLogKubernetesAPIError(err) {
		return ctx.Log().ErrorfNewErr("Failed creating/updating Prometheus auth policy: %v", err)
	}
	return err
}

// createOrUpdateServiceMonitors creates or updates service monitors to meet operator requirements
func createOrUpdateServiceMonitors(ctx spi.ComponentContext) error {
	smList := &promoperapi.ServiceMonitorList{}
	enableHTTP2 := false
	err := ctx.Client().List(context.TODO(), smList)
	if err != nil {
		return err
	}
	for _, sm := range smList.Items {
		var endPoints []promoperapi.Endpoint
		for _, ep := range sm.Spec.Endpoints {
			if ep.Scheme == "https" {
				ep.EnableHttp2 = &enableHTTP2
			}
			endPoints = append(endPoints, ep)
		}
		sm.Spec.Endpoints = endPoints
		smSpec := sm.Spec.DeepCopy()
		_, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client(), sm, func() error {
			sm.Spec = promoperapi.ServiceMonitorSpec{
				JobLabel:              smSpec.JobLabel,
				TargetLabels:          smSpec.TargetLabels,
				PodTargetLabels:       smSpec.PodTargetLabels,
				Endpoints:             smSpec.Endpoints,
				Selector:              smSpec.Selector,
				NamespaceSelector:     smSpec.NamespaceSelector,
				SampleLimit:           smSpec.SampleLimit,
				TargetLimit:           smSpec.TargetLimit,
				LabelLimit:            smSpec.LabelLimit,
				LabelNameLengthLimit:  smSpec.LabelNameLengthLimit,
				LabelValueLengthLimit: smSpec.LabelValueLengthLimit,
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// reconcileIngresses reconciles ingresses for the Prometheus and Alertmanager endpoint
func reconcileIngresses(ctx spi.ComponentContext) error {
	// If NGINX is not enabled, skip the ingress creation and delete any existing prometheus or alertmanager ingresses
	if !vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {
		return deletePrometheusStackIngresses(ctx)
	}

	promProps := common.IngressProperties{
		IngressName:   constants.PrometheusIngress,
		HostName:      prometheusHostName,
		TLSSecretName: prometheusCertificateName,
		// Enable sticky sessions, so there is no UI query skew in multi-replica prometheus clusters
		ExtraAnnotations: common.SameSiteCookieAnnotations(prometheusName),
	}
	err := common.CreateOrUpdateSystemComponentIngress(ctx, promProps)
	if err != nil {
		return err
	}

	return reconcileAlertmanagerIngress(ctx)
}

// deleteNetworkPolicy deletes the existing NetworkPolicy. Since the NetworkPolicy is now part of the Helm chart,
// upgrading from a previous installation fails if the NetworkPolicy already exists and is not owned by the Helm chart,
// so we delete it before upgrading.
func deleteNetworkPolicy(ctx spi.ComponentContext) error {
	netpol := &netv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: networkPolicyName, Namespace: ComponentNamespace}}
	err := client.IgnoreNotFound(ctx.Client().Delete(context.TODO(), netpol))
	if err != nil {
		ctx.Log().Errorf("Error deleting existing NetworkPolicy %s/%s: %v", networkPolicyName, ComponentNamespace, err)
		return err
	}
	return nil
}

// reconcileAlertmanagerIngress reconciles the ingress for Alertmanager endpoint
func reconcileAlertmanagerIngress(ctx spi.ComponentContext) error {
	alertmanagerEnabled, err := vzcr.IsAlertmanagerEnabled(ctx.EffectiveCR(), ctx)
	if err != nil {
		return err
	}

	if alertmanagerEnabled {
		alertmanagerProps := common.IngressProperties{
			IngressName:   constants.AlertmanagerIngress,
			HostName:      alertmanagerHostName,
			TLSSecretName: alertmanagerCertificateName,
		}
		err := common.CreateOrUpdateSystemComponentIngress(ctx, alertmanagerProps)
		if err != nil {
			return err
		}
	} else {
		return deleteIngress(constants.AlertmanagerIngress, ctx)
	}

	return nil
}

func createOrUpdateAlertmanagerConfig(ctx spi.ComponentContext) error {
	alertmanagerEnabled, err := vzcr.IsAlertmanagerEnabled(ctx.EffectiveCR(), ctx)
	if err != nil {
		return err
	}

	if alertmanagerEnabled {
		alertmanagerConfig := &promoperapiv1alpha1.AlertmanagerConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      alertmanagerConfigName,
				Namespace: ComponentNamespace,
				Labels: map[string]string{
					alertmanagerConfigLabelName: alertmanagerConfigLabelValue,
				},
			},
		}
		_, err = controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), alertmanagerConfig, func() error {
			if alertmanagerConfig.Spec.Receivers == nil || len(alertmanagerConfig.Spec.Receivers) == 0 {
				alertmanagerConfig.Spec.Receivers = []promoperapiv1alpha1.Receiver{
					{
						Name: nullAlertmanagerConfigReceiver,
					},
				}
			}
			if alertmanagerConfig.Spec.Route == nil {
				alertmanagerConfig.Spec.Route = &promoperapiv1alpha1.Route{
					Receiver:       nullAlertmanagerConfigReceiver,
					GroupWait:      "0s",
					GroupInterval:  "5m",
					RepeatInterval: "4h",
				}
			}
			return nil
		})
		return err
	}
	return nil
}
