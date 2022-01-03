// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cohworkload

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/logging"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	istionet "istio.io/api/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const cohFluentdParsingRules = `<match fluent.**>
@type null
</match>

# Coherence Logs
<source>                                    
@type tail
path /logs/coherence-*.log
pos_file /tmp/cohrence.log.pos
read_from_head true
tag coherence-cluster
multiline_flush_interval 20s
<parse>
 @type multiline
 format_firstline /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3}/
 format1 /^(?<time>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3})\/(?<uptime>[0-9\.]+) (?<product>.+) <(?<level>[^\s]+)> \(thread=(?<thread>.+), member=(?<member>.+)\):[\S\s](?<log>.*)/
</parse>
</source>

<filter coherence-cluster>                  
@type record_transformer
<record>
 coherence.cluster.name "#{ENV['COH_CLUSTER_NAME']}"
 role "#{ENV['COH_ROLE']}"
 host "#{ENV['HOSTNAME']}"
 pod-uid "#{ENV['COH_POD_UID']}"
 oam.applicationconfiguration.namespace "#{ENV['NAMESPACE']}"
 oam.applicationconfiguration.name "#{ENV['APP_CONF_NAME']}"
 oam.component.namespace "#{ENV['NAMESPACE']}"
 oam.component.name  "#{ENV['COMPONENT_NAME']}"
 verrazzano.cluster.name  "#{ENV['CLUSTER_NAME']}"
</record>
</filter>

<match coherence-cluster>
  @type stdout
</match>
`

const (
	specField                 = "spec"
	jvmField                  = "jvm"
	argsField                 = "args"
	workloadType              = "coherence"
	destinationRuleAPIVersion = "networking.istio.io/v1alpha3"
	destinationRuleKind       = "DestinationRule"
	coherenceExtendPort       = 9000
	loggingNamePart           = "logging-stdout"
	loggingMountPath          = "/fluentd/etc/custom.conf"
	loggingKey                = "custom.conf"
	fluentdVolumeName         = "fluentd-config-volume"
)

var specLabelsFields = []string{specField, "labels"}
var specAnnotationsFields = []string{specField, "annotations"}

// additional JVM args that need to get added to the Coherence spec to enable logging
var additionalJvmArgs = []interface{}{
	"-Dcoherence.log=jdk",
	"-Dcoherence.log.logger=com.oracle.coherence",
	"-Djava.util.logging.config.file=/coherence-operator/utils/logging/logging.properties",
}

// this struct allows us to extract information from the unstructured Coherence spec
// so we can interface with the FLUENTD code
type containersMountsVolumes struct {
	SideCars     []corev1.Container
	Volumes      []corev1.Volume
	VolumeMounts []corev1.VolumeMount
}

// Reconciler reconciles a VerrazzanoCoherenceWorkload object
type Reconciler struct {
	client.Client
	Log     logr.Logger
	Scheme  *runtime.Scheme
	Metrics *metricstrait.Reconciler
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vzapi.VerrazzanoCoherenceWorkload{}).
		Complete(r)
}

// Reconcile reconciles a VerrazzanoCoherenceWorkload resource. It fetches the embedded Coherence CR, mutates it to add
// scopes and traits, and then writes out the CR (or deletes it if the workload is being deleted).
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanocoherenceworkloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanocoherenceworkloads/status,verbs=get;update;patch
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("verrazzanocoherenceworkload", req.NamespacedName)
	log.Info("Reconciling Verrazzano Coherence workload")

	// fetch the workload and unwrap the Coherence resource
	workload, err := r.fetchWorkload(ctx, req.NamespacedName)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// Make sure the last generation exists in the status
	result, err := r.ensureLastGeneration(workload)
	if err != nil || result.Requeue {
		return result, err
	}

	u, err := vznav.ConvertRawExtensionToUnstructured(&workload.Spec.Template)
	if err != nil {
		return reconcile.Result{}, err
	}

	// make sure the namespace is set to the namespace of the component
	if err = unstructured.SetNestedField(u.Object, req.NamespacedName.Namespace, "metadata", "namespace"); err != nil {
		return reconcile.Result{}, err
	}

	// the embedded resource doesn't have an API version or kind, so add them
	gvk := vznav.APIVersionAndKindToContainedGVK(workload.APIVersion, workload.Kind)
	if gvk == nil {
		errStr := "unable to determine contained GroupVersionKind for workload"
		log.Error(nil, errStr, "workload", workload)
		return reconcile.Result{}, errors.New(errStr)
	}

	apiVersion, kind := gvk.ToAPIVersionAndKind()
	u.SetAPIVersion(apiVersion)
	u.SetKind(kind)

	// mutate the Coherence resource, copy labels, add logging, etc.
	if err = copyLabels(log, workload.ObjectMeta.GetLabels(), u); err != nil {
		return reconcile.Result{}, err
	}

	spec, found, _ := unstructured.NestedMap(u.Object, specField)
	if !found {
		return reconcile.Result{}, errors.New("embedded Coherence resource is missing the required 'spec' field")
	}

	// Attempt to get the existing Coherence StatefulSet. This is used in the case where we don't want to update any resources
	// which are defined by Verrazzano such as the Fluentd image used by logging. In this case we obtain the previous
	// Fluentd image and set that on the new Coherence StatefulSet.
	var existingCoherence v1.StatefulSet
	domainExists := true
	coherenceKey := types.NamespacedName{Name: u.GetName(), Namespace: workload.Namespace}
	if err := r.Get(ctx, coherenceKey, &existingCoherence); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("No existing Coherence StatefulSet found")
			domainExists = false
		} else {
			log.Error(err, "An error occurred trying to obtain an existing Coherence StatefulSet")
			return reconcile.Result{}, err
		}
	}

	// If the Coherence cluster already exists, make sure that it can be restarted.
	// If the cluster cannot be restarted, don't make any Coherence changes.
	if domainExists && !r.isOkToRestartCoherence(workload) {
		log.Info("There have been no changes to the Coherence workload, nor has the restart annotation changed. The Coherence resource will not be modified.")
		return ctrl.Result{}, nil
	}

	// Add the Fluentd sidecar container required for logging to the Coherence StatefulSet
	if err = r.addLogging(ctx, log, workload, spec, &existingCoherence); err != nil {
		return reconcile.Result{}, err
	}

	// Add logging traits to the Domain if they exist
	if err = r.addLoggingTrait(ctx, log, workload, u, spec); err != nil {
		return reconcile.Result{}, err
	}

	// spec has been updated with logging, set the new entries in the unstructured
	if err = unstructured.SetNestedField(u.Object, spec, specField); err != nil {
		return reconcile.Result{}, err
	}

	if err = r.addMetrics(ctx, log, req.NamespacedName.Namespace, workload, u); err != nil {
		return reconcile.Result{}, err
	}

	// set istio injection annotation to false for Coherence pods
	if err = r.disableIstioInjection(log, u); err != nil {
		return reconcile.Result{}, err
	}

	// set controller reference so the Coherence CR gets deleted when the workload is deleted
	if err = controllerutil.SetControllerReference(workload, u, r.Scheme); err != nil {
		log.Error(err, "Unable to set controller ref")
		return reconcile.Result{}, err
	}

	// write out restart-version in Coherence spec annotations
	cohName, _, err := unstructured.NestedString(u.Object, "metadata", "name")
	if err != nil {
		return reconcile.Result{}, err
	}
	if err = r.addRestartVersionAnnotation(u, workload.Annotations[vzconst.RestartVersionAnnotation], cohName, workload.Namespace, log); err != nil {
		return reconcile.Result{}, err
	}

	// make a copy of the Coherence spec since u.Object will get overwritten in CreateOrUpdate
	// if the Coherence CR exists
	specCopy, _, err := unstructured.NestedFieldCopy(u.Object, specField)
	if err != nil {
		log.Error(err, "Unable to make a copy of the Coherence spec")
		return reconcile.Result{}, err
	}

	// write out the Coherence resource
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, u, func() error {
		return unstructured.SetNestedField(u.Object, specCopy, specField)
	})
	if err != nil {
		log.Error(err, "Error creating or updating Coherence CR")
		return reconcile.Result{}, err
	}

	// Get the namespace resource that the VerrazzanoCoherenceWorkload resource is deployed to
	namespace := &corev1.Namespace{}
	if err = r.Client.Get(ctx, client.ObjectKey{Namespace: "", Name: req.NamespacedName.Namespace}, namespace); err != nil {
		return reconcile.Result{}, err
	}

	if err = r.createOrUpdateDestinationRule(ctx, log, namespace.Name, namespace.Labels, workload.ObjectMeta.Labels); err != nil {
		return reconcile.Result{}, err
	}

	if err = r.updateStatusReconcileDone(ctx, workload); err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Successfully reconciled Verrazzano Coherence workload")
	return reconcile.Result{}, nil
}

// fetchWorkload fetches the VerrazzanoCoherenceWorkload data given a namespaced name
func (r *Reconciler) fetchWorkload(ctx context.Context, name types.NamespacedName) (*vzapi.VerrazzanoCoherenceWorkload, error) {
	var workload vzapi.VerrazzanoCoherenceWorkload
	if err := r.Get(ctx, name, &workload); err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Info("VerrazzanoCoherenceWorkload has been deleted", "workload", name)
		} else {
			r.Log.Error(err, "Failed to fetch VerrazzanoCoherenceWorkload", "workload", name)
		}
		return nil, err
	}

	return &workload, nil
}

// copyLabels copies specific labels from the Verrazzano workload to the contained Coherence resource
func copyLabels(log logr.Logger, workloadLabels map[string]string, coherence *unstructured.Unstructured) error {
	labels, found, _ := unstructured.NestedStringMap(coherence.Object, specLabelsFields...)
	if !found {
		labels = map[string]string{}
	}

	// copy the oam component and app name labels
	if componentName, ok := workloadLabels[oam.LabelAppComponent]; ok {
		labels[oam.LabelAppComponent] = componentName
	}

	if appName, ok := workloadLabels[oam.LabelAppName]; ok {
		labels[oam.LabelAppName] = appName
	}

	err := unstructured.SetNestedStringMap(coherence.Object, labels, specLabelsFields...)
	if err != nil {
		log.Error(err, "Unable to set labels in spec")
		return err
	}

	return nil
}

// disableIstioInjection sets the sidecar.istio.io/inject annotation to false since Coherence does not work with Istio
func (r *Reconciler) disableIstioInjection(log logr.Logger, u *unstructured.Unstructured) error {
	annotations, _, err := unstructured.NestedStringMap(u.Object, specAnnotationsFields...)
	if err != nil {
		return errors.New("unable to get annotations from Coherence spec")
	}

	// if no annotations exist initialize the annotations map otherwise update existing annotations.
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["sidecar.istio.io/inject"] = "false"

	err = unstructured.SetNestedStringMap(u.Object, annotations, specAnnotationsFields...)
	if err != nil {
		return err
	}

	return nil
}

// addLogging adds a FLUENTD sidecar and updates the Coherence spec if there is an associated LogInfo
func (r *Reconciler) addLogging(ctx context.Context, log logr.Logger, workload *vzapi.VerrazzanoCoherenceWorkload, coherenceSpec map[string]interface{}, existingCoherence *v1.StatefulSet) error {
	// extract just enough of the Coherence data into concrete types so we can merge with
	// the FLUENTD data
	var extracted containersMountsVolumes
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(coherenceSpec, &extracted); err != nil {
		return errors.New("unable to extract containers, volumes, and volume mounts from Coherence spec")
	}

	// fluentdPod starts with what's in the spec and we add in the FLUENTD things when Apply is
	// called on the fluentdManager
	fluentdPod := &logging.FluentdPod{
		Containers:   extracted.SideCars,
		Volumes:      extracted.Volumes,
		VolumeMounts: extracted.VolumeMounts,
		LogPath:      "/logs",
	}
	fluentdManager := &logging.Fluentd{
		Context:                ctx,
		Log:                    log,
		Client:                 r.Client,
		ParseRules:             cohFluentdParsingRules,
		StorageVolumeName:      "logs",
		StorageVolumeMountPath: "/logs",
		WorkloadType:           workloadType,
	}

	// fluentdManager.Apply wants a QRR but it only cares about the namespace (at least for
	// this use case)
	resource := vzapi.QualifiedResourceRelation{Namespace: workload.Namespace}

	// note that this call has the side effect of creating a FLUENTD config map if one
	// does not already exist in the namespace
	if _, err := fluentdManager.Apply(logging.NewLogInfo(), resource, fluentdPod); err != nil {
		return err
	}

	// fluentdPod now has the FLUENTD container, volumes, and volume mounts merged in
	// with the existing spec data

	// Coherence wants the volume mount for the FLUENTD config map stored in "configMapVolumes", so
	// we have to move it from the FLUENTD container volume mounts
	if err := moveConfigMapVolume(log, fluentdPod, coherenceSpec); err != nil {
		return err
	}

	// convert the containers, volumes, and mounts in fluentdPod to unstructured and set
	// the values in the spec
	fluentdPodUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(fluentdPod)
	if err != nil {
		return err
	}

	coherenceSpec["sideCars"] = fluentdPodUnstructured["containers"]
	coherenceSpec["volumes"] = fluentdPodUnstructured["volumes"]
	coherenceSpec["volumeMounts"] = fluentdPodUnstructured["volumeMounts"]

	addJvmArgs(coherenceSpec)

	return nil
}

// addMetrics adds the labels and annotations needed for metrics to the Coherence resource annotations which are propagated to the individual Coherence pods.
// Returns the success fo the operation and any error occurred. If metrics were successfully added, true is return with a nil error.
func (r *Reconciler) addMetrics(ctx context.Context, log logr.Logger, namespace string, workload *vzapi.VerrazzanoCoherenceWorkload, coherence *unstructured.Unstructured) error {
	log.Info(fmt.Sprintf("Adding Metrics for: %s", workload.Name))
	metricsTrait, err := vznav.MetricsTraitFromWorkloadLabels(ctx, r.Client, log, namespace, workload.ObjectMeta)
	if err != nil {
		return err
	}

	if metricsTrait == nil {
		log.Info("Workload has no associated MetricTrait, nothing to do")
		return nil
	}
	log.Info(fmt.Sprintf("Found associated metrics trait for workload: %s : %s", workload.Name, metricsTrait.Name))

	traitDefaults, err := r.Metrics.NewTraitDefaultsForCOHWorkload(ctx, coherence)
	if err != nil {
		log.Error(err, "Unable to get default metric trait values")
		return err
	}

	metricAnnotations, found, _ := unstructured.NestedStringMap(coherence.Object, specAnnotationsFields...)
	if !found {
		metricAnnotations = map[string]string{}
	}

	metricLabels, found, _ := unstructured.NestedStringMap(coherence.Object, specLabelsFields...)
	if !found {
		metricLabels = map[string]string{}
	}

	finalAnnotations := metricstrait.MutateAnnotations(metricsTrait, coherence, traitDefaults, metricAnnotations)
	log.Info(fmt.Sprintf("Setting annotations on %s: %v", workload.Name, finalAnnotations))
	err = unstructured.SetNestedStringMap(coherence.Object, finalAnnotations, specAnnotationsFields...)
	if err != nil {
		log.Error(err, "Unable to set metric annotations on Coherence resource")
		return err
	}

	finalLabels := metricstrait.MutateLabels(metricsTrait, coherence, metricLabels)
	log.Info(fmt.Sprintf("Setting labels on %s: %v", workload.Name, finalLabels))

	err = unstructured.SetNestedStringMap(coherence.Object, finalLabels, specLabelsFields...)
	if err != nil {
		log.Error(err, "Unable to set metric labels on Coherence resource")
		return err
	}

	return nil
}

// moveConfigMapVolume moves the FLUENTD config map volume definition. Coherence wants the volume mount
// for the FLUENTD config map stored in "configMapVolumes", so we will pull the mount out from the
// FLUENTD container and put it in its new home in the Coherence spec (this should all be handled
// by the FLUENTD code at some point but I tried to limit the surgery for now)
func moveConfigMapVolume(log logr.Logger, fluentdPod *logging.FluentdPod, coherenceSpec map[string]interface{}) error {
	var fluentdVolMount corev1.VolumeMount

	for _, container := range fluentdPod.Containers {
		if container.Name == logging.FluentdStdoutSidecarName {
			fluentdVolMount = container.VolumeMounts[0]
			// Coherence needs the vol mount to match the config map name, so fix it, need
			// to see if we can just change name set by the FLUENTD code
			fluentdVolMount.Name = "fluentd-config" + "-" + workloadType
			fluentdPod.Containers[0].VolumeMounts = nil
			break
		}
	}

	// add the config map volume mount to "configMapVolumes" in the spec
	if fluentdVolMount.Name != "" {
		fluentdVolMountUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&fluentdVolMount)
		if err != nil {
			return err
		}

		if configMapVolumes, found := coherenceSpec["configMapVolumes"]; !found {
			coherenceSpec["configMapVolumes"] = []interface{}{fluentdVolMountUnstructured}
		} else {
			vols := configMapVolumes.([]interface{})
			coherenceSpec["configMapVolumes"] = append(vols, fluentdVolMountUnstructured)
		}
	} else {
		log.Info("Expected to find config map volume mount in fluentd container but did not")
	}

	volumes := fluentdPod.Volumes
	vIndex := -1
	for v, volume := range volumes {
		if volume.Name == fluentdVolumeName {
			vIndex = v
		}
	}
	if vIndex != -1 {
		volumes[vIndex] = volumes[len(volumes)-1]
		fluentdPod.Volumes = volumes[:len(volumes)-1]
	}

	return nil
}

// addJvmArgs adds the additional JVM args needed to enable and configure logging
// in the Coherence container
func addJvmArgs(coherenceSpec map[string]interface{}) {
	var jvm map[string]interface{}
	if val, found := coherenceSpec[jvmField]; !found {
		jvm = make(map[string]interface{})
		coherenceSpec[jvmField] = jvm
	} else {
		jvm = val.(map[string]interface{})
	}

	var args []interface{}
	if val, found := jvm[argsField]; !found {
		args = additionalJvmArgs
	} else {
		// just append our logging args, this needs to be improved to handle
		// the case where one or more of the args are already present
		args = val.([]interface{})
		args = append(args, additionalJvmArgs...)
	}
	jvm[argsField] = args
}

// createOrUpdateDestinationRule creates or updates an Istio destinationRule required by Coherence.
// The destinationRule is only created when the namespace has the label istio-injection=enabled.
func (r *Reconciler) createOrUpdateDestinationRule(ctx context.Context, log logr.Logger, namespace string, namespaceLabels map[string]string, workloadLabels map[string]string) error {
	istioEnabled := false
	value, ok := namespaceLabels["istio-injection"]
	if ok && value == "enabled" {
		istioEnabled = true
	}

	if !istioEnabled {
		return nil
	}

	appName, ok := workloadLabels[oam.LabelAppName]
	if !ok {
		return errors.New("OAM app name label missing from metadata, unable to generate destination rule name")
	}

	// Create a destinationRule populating only name metadata.
	// This is used as default if the destinationRule needs to be created.
	destinationRule := &istioclient.DestinationRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: destinationRuleAPIVersion,
			Kind:       destinationRuleKind},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      appName,
		},
	}

	log.Info(fmt.Sprintf("Creating/updating destination rule %s:%s", namespace, appName))
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, destinationRule, func() error {
		return r.mutateDestinationRule(destinationRule, namespace, appName)
	})

	return err
}

// mutateDestinationRule mutates the output destinationRule.
func (r *Reconciler) mutateDestinationRule(destinationRule *istioclient.DestinationRule, namespace string, appName string) error {
	// Set the spec content.
	destinationRule.Spec.Host = fmt.Sprintf("*.%s.svc.cluster.local", namespace)
	destinationRule.Spec.TrafficPolicy = &istionet.TrafficPolicy{
		Tls: &istionet.ClientTLSSettings{
			Mode: istionet.ClientTLSSettings_ISTIO_MUTUAL,
		},
	}
	destinationRule.Spec.TrafficPolicy.PortLevelSettings = []*istionet.TrafficPolicy_PortTrafficPolicy{
		{
			// Disable mutual TLS for the Coherence extend port
			Port: &istionet.PortSelector{
				Number: coherenceExtendPort,
			},
			Tls: &istionet.ClientTLSSettings{
				Mode: istionet.ClientTLSSettings_DISABLE,
			},
		},
	}

	// Set the owner reference.
	appConfig := &v1alpha2.ApplicationConfiguration{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: appName}, appConfig)
	if err != nil {
		return err
	}
	err = controllerutil.SetControllerReference(appConfig, destinationRule, r.Scheme)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) updateStatusReconcileDone(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload) error {
	if workload.Status.LastGeneration != strconv.Itoa(int(workload.Generation)) {
		workload.Status.LastGeneration = strconv.Itoa(int(workload.Generation))
		return r.Status().Update(ctx, workload)
	}
	return nil
}

// addLoggingTrait adds the logging trait sidecar to the workload
func (r *Reconciler) addLoggingTrait(ctx context.Context, log logr.Logger, workload *vzapi.VerrazzanoCoherenceWorkload, coherence *unstructured.Unstructured, coherenceSpec map[string]interface{}) error {
	loggingTrait, err := vznav.LoggingTraitFromWorkloadLabels(ctx, r.Client, log, workload.GetNamespace(), workload.ObjectMeta)
	if err != nil {
		return err
	}
	if loggingTrait == nil {
		return nil
	}

	configMapName := loggingNamePart + "-" + coherence.GetName() + "-" + strings.ToLower(coherence.GetKind())
	configMap := &corev1.ConfigMap{}
	err = r.Get(ctx, client.ObjectKey{Namespace: coherence.GetNamespace(), Name: configMapName}, configMap)
	if err != nil && k8serrors.IsNotFound(err) {
		data := make(map[string]string)
		data["custom.conf"] = loggingTrait.Spec.LoggingConfig
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      loggingNamePart + "-" + coherence.GetName() + "-" + strings.ToLower(coherence.GetKind()),
				Namespace: coherence.GetNamespace(),
				Labels:    coherence.GetLabels(),
			},
			Data: data,
		}
		err = controllerutil.SetControllerReference(workload, configMap, r.Scheme)
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("Creating logging trait configmap %s:%s", coherence.GetNamespace(), configMapName))
		err = r.Create(ctx, configMap)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("logging trait configmap %s:%s already exist", coherence.GetNamespace(), configMapName))

	// extract just enough of the WebLogic data into concrete types so we can merge with
	// the logging trait data
	var extract containersMountsVolumes
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(coherenceSpec, &extract); err != nil {
		return errors.New("unable to extract containers, volumes, and volume mounts from Coherence spec")
	}
	extracted := &containersMountsVolumes{
		SideCars:     extract.SideCars,
		VolumeMounts: extract.VolumeMounts,
		Volumes:      extract.Volumes,
	}
	loggingVolumeMount := &corev1.VolumeMount{
		MountPath: loggingMountPath,
		Name:      configMapName,
		SubPath:   loggingKey,
		ReadOnly:  true,
	}

	loggingVolumeMountUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&loggingVolumeMount)
	if err != nil {
		return err
	}
	if configMapVolumes, found := coherenceSpec["configMapVolumes"]; !found {
		coherenceSpec["configMapVolumes"] = []interface{}{loggingVolumeMountUnstructured}
	} else {
		vols := configMapVolumes.([]interface{})
		volIndex := -1
		for i, v := range vols {
			if v.(map[string]interface{})["mountPath"] == loggingVolumeMountUnstructured["mountPath"] && v.(map[string]interface{})["name"] == loggingVolumeMountUnstructured["name"] {
				volIndex = i
			}
		}
		if volIndex == -1 {
			vols = append(vols, loggingVolumeMountUnstructured)
		} else {
			vols[volIndex] = loggingVolumeMountUnstructured
		}
		coherenceSpec["configMapVolumes"] = vols
	}
	var image string
	if len(loggingTrait.Spec.LoggingImage) != 0 {
		image = loggingTrait.Spec.LoggingImage
	} else {
		image = os.Getenv("DEFAULT_FLUENTD_IMAGE")
	}
	envFluentd := &corev1.EnvVar{
		Name:  "FLUENTD_CONF",
		Value: "custom.conf",
	}
	loggingContainer := &corev1.Container{
		Name:            loggingNamePart,
		Image:           image,
		ImagePullPolicy: corev1.PullPolicy(loggingTrait.Spec.ImagePullPolicy),
		Env:             []corev1.EnvVar{*envFluentd},
	}
	sIndex := -1
	for i, s := range extracted.SideCars {
		if s.Name == loggingNamePart {
			sIndex = i
		}
	}
	if sIndex != -1 {
		extracted.SideCars[sIndex] = *loggingContainer
	} else {
		extracted.SideCars = append(extracted.SideCars, *loggingContainer)
	}

	// convert the containers, volumes, and mounts in extracted to unstructured and set
	// the values in the spec
	extractedUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&extracted)
	if err != nil {
		return err
	}
	coherenceSpec["sideCars"] = extractedUnstructured["sideCars"]

	return nil
}

func (r *Reconciler) addRestartVersionAnnotation(coherence *unstructured.Unstructured, restartVersion string, name, namespace string, log logr.Logger) error {
	if len(restartVersion) > 0 {
		log.Info(fmt.Sprintf("The Coherence %s/%s restart version is set to %s", namespace, name, restartVersion))
		annotations, _, err := unstructured.NestedStringMap(coherence.Object, specAnnotationsFields...)
		if err != nil {
			return errors.New("unable to get annotations from Coherence spec")
		}
		// if no annotations exist initialize the annotations map otherwise update existing annotations.
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[vzconst.RestartVersionAnnotation] = restartVersion
		return unstructured.SetNestedStringMap(coherence.Object, annotations, specAnnotationsFields...)
	}
	return nil
}

// Make sure that the last generation exists in the status
func (r *Reconciler) ensureLastGeneration(wl *vzapi.VerrazzanoCoherenceWorkload) (ctrl.Result, error) {
	if len(wl.Status.LastGeneration) > 0 {
		return ctrl.Result{}, nil
	}

	// Update the status generation and always requeue
	wl.Status.LastGeneration = strconv.Itoa(int(wl.Generation))
	err := r.Status().Update(context.TODO(), wl)
	return ctrl.Result{Requeue: true, RequeueAfter: 1}, err
}

// Make sure that it is OK to restart Coherence
func (r *Reconciler) isOkToRestartCoherence(coh *vzapi.VerrazzanoCoherenceWorkload) bool {
	// Check if user created or changed the restart annotation
	if coh.Annotations != nil && coh.Annotations[vzconst.RestartVersionAnnotation] != coh.Status.LastRestartVersion {
		return true
	}
	if coh.Status.LastGeneration == strconv.Itoa(int(coh.Generation)) {
		// nothing in the spec has changed
		return false
	}
	// The spec has changed because the generation is different from the saved one
	return true
}
