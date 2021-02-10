// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cohworkload

import (
	"context"
	"errors"

	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
	"github.com/verrazzano/verrazzano/application-operator/controllers/loggingscope"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const fluentdParsingRules = `<match fluent.**>
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
 cluster "#{ENV['COH_CLUSTER_NAME']}"
 role "#{ENV['COH_ROLE']}"
 host "#{ENV['HOSTNAME']}"
 pod-uid "#{ENV['COH_POD_UID']}"
 oam.applicationconfiguration.namespace "#{ENV['NAMESPACE']}"
 oam.applicationconfiguration.name "#{ENV['APP_CONF_NAME']}"
 oam.component.namespace "#{ENV['NAMESPACE']}"
 oam.component.name  "#{ENV['COMPONENT_NAME']}"
</record>
</filter>

<match coherence-cluster>
  @type elasticsearch
  host "#{ENV['ELASTICSEARCH_HOST']}"
  port "#{ENV['ELASTICSEARCH_PORT']}"
  user "#{ENV['ELASTICSEARCH_USER']}"
  password "#{ENV['ELASTICSEARCH_PASSWORD']}"
  index_name "` + loggingscope.ElasticSearchIndex + `"
  scheme http
  key_name timestamp 
  types timestamp:time
  include_timestamp true
</match>
`

const finalizer = "verrazzanocoherenceworkload.finalizers.verrazzano.io"

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
	Log    logr.Logger
	Scheme *runtime.Scheme
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
	log.Info("Reconciling verrazzano coherence workload")

	// fetch the workload and unwrap the Coherence resource
	workload, err := r.fetchWorkload(ctx, req.NamespacedName)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
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
		errStr := "Unable to determine contained GroupVersionKind for workload"
		log.Error(nil, errStr, "workload", workload)
		return reconcile.Result{}, errors.New(errStr)
	}

	apiVersion, kind := gvk.ToAPIVersionAndKind()
	u.SetAPIVersion(apiVersion)
	u.SetKind(kind)

	// clean up resources that we've created on delete
	if isDeleting, err := r.handleDelete(ctx, log, workload, u); isDeleting {
		return reconcile.Result{}, err
	}

	// mutate the Coherence resource, copy labels, add logging, etc.
	if err = copyLabels(log, workload.ObjectMeta.GetLabels(), u); err != nil {
		return reconcile.Result{}, err
	}

	spec, found, _ := unstructured.NestedMap(u.Object, "spec")
	if !found {
		return reconcile.Result{}, errors.New("Embedded Coherence resource is missing the required 'spec' field")
	}

	if err = r.addLogging(ctx, log, req.NamespacedName.Namespace, workload.ObjectMeta.Labels, spec); err != nil {
		return reconcile.Result{}, err
	}

	// spec has been updated with logging, need to set it back in the unstructured
	if err = unstructured.SetNestedField(u.Object, spec, "spec"); err != nil {
		return reconcile.Result{}, err
	}

	// set istio injection annotation to false for Coherence pods
	if err = r.disableIstioInjection(log, u); err != nil {
		return reconcile.Result{}, err
	}

	// write out the Coherence resource
	if err = r.Client.Create(ctx, u); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return reconcile.Result{}, err
		}
		log.Info("Coherence CR already exists, ignoring error on create")
		return reconcile.Result{}, nil
	}

	log.Info("Successfully created Verrazzano Coherence workload")
	return reconcile.Result{}, nil
}

// fetchWorkload fetches the VerrazzanoCoherenceWorkload data given a namespaced name
func (r *Reconciler) fetchWorkload(ctx context.Context, name types.NamespacedName) (*vzapi.VerrazzanoCoherenceWorkload, error) {
	var workload vzapi.VerrazzanoCoherenceWorkload
	if err := r.Get(ctx, name, &workload); err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Info("VerrazzanoCoherenceWorkload has been deleted", "name", name)
		} else {
			r.Log.Error(err, "Failed to fetch VerrazzanoCoherenceWorkload", "name", name)
		}
		return nil, err
	}

	return &workload, nil
}

// copyLabels copies specific labels from the Verrazzano workload to the contained Coherence resource
func copyLabels(log logr.Logger, workloadLabels map[string]string, coherence *unstructured.Unstructured) error {
	labels, found, _ := unstructured.NestedStringMap(coherence.Object, "spec", "labels")
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

	err := unstructured.SetNestedStringMap(coherence.Object, labels, "spec", "labels")
	if err != nil {
		log.Error(err, "Unable to set labels in spec")
		return err
	}

	return nil
}

// disableIstioInjection sets the sidecar.istio.io/inject annotation to false since Coherence does not work with Istio
func (r *Reconciler) disableIstioInjection(log logr.Logger, u *unstructured.Unstructured) error {
	annotations, _, err := unstructured.NestedStringMap(u.Object, "spec", "annotations")
	if err != nil {
		return errors.New("unable to get annotations from Coherence spec")
	}

	// if no annotations exist initialize the annotations map otherwise update existing annotations.
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["sidecar.istio.io/inject"] = "false"

	err = unstructured.SetNestedStringMap(u.Object, annotations, "spec", "annotations")
	if err != nil {
		return err
	}

	return nil
}

// addLogging adds a FLUENTD sidecar and updates the Coherence spec if there is an associated LoggingScope
func (r *Reconciler) addLogging(ctx context.Context, log logr.Logger, namespace string, labels map[string]string, coherenceSpec map[string]interface{}) error {
	loggingScope, err := vznav.LoggingScopeFromWorkloadLabels(ctx, r.Client, namespace, labels)
	if err != nil {
		return err
	}

	if loggingScope == nil {
		log.Info("No logging scope found for workload, nothing to do")
		return nil
	}

	// extract just enough of the Coherence data into concrete types so we can merge with
	// the FLUENTD data
	var extracted containersMountsVolumes
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(coherenceSpec, &extracted); err != nil {
		return errors.New("Unable to extract containers, volumes, and volume mounts from Coherence spec")
	}

	// fluentdPod starts with what's in the spec and we add in the FLUENTD things when Apply is
	// called on the fluentdManager
	fluentdPod := &loggingscope.FluentdPod{
		Containers:   extracted.SideCars,
		Volumes:      extracted.Volumes,
		VolumeMounts: extracted.VolumeMounts,
		LogPath:      "/logs",
	}
	fluentdManager := &loggingscope.Fluentd{
		Context:                ctx,
		Log:                    log,
		Client:                 r.Client,
		ParseRules:             fluentdParsingRules,
		StorageVolumeName:      "logs",
		StorageVolumeMountPath: "/logs",
	}

	// fluentdManager.Apply wants a QRR but it only cares about the namespace (at least for
	// this use case)
	resource := vzapi.QualifiedResourceRelation{Namespace: namespace}

	// note that this call has the side effect of creating a FLUENTD config map if one
	// does not already exist in the namespace
	if _, err = fluentdManager.Apply(loggingScope, resource, fluentdPod); err != nil {
		return err
	}

	// fluentdPod now has the FLUENTD container, volumes, and volume mounts merged in
	// with the existing spec data

	// Coherence wants the volume mount for the FLUENTD config map stored in "configMapVolumes", so
	// we have to move it from the FLUENTD container volume mounts
	if err = moveConfigMapVolume(log, fluentdPod, coherenceSpec); err != nil {
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

// moveConfigMapVolume moves the FLUENTD config map volume definition. Coherence wants the volume mount
// for the FLUENTD config map stored in "configMapVolumes", so we will pull the mount out from the
// FLUENTD container and put it in its new home in the Coherence spec (this should all be handled
// by the FLUENTD code at some point but I tried to limit the surgery for now)
func moveConfigMapVolume(log logr.Logger, fluentdPod *loggingscope.FluentdPod, coherenceSpec map[string]interface{}) error {
	var fluentdVolMount corev1.VolumeMount

	for _, container := range fluentdPod.Containers {
		if container.Name == "fluentd" {
			fluentdVolMount = container.VolumeMounts[0]
			// Coherence needs the vol mount to match the config map name, so fix it, need
			// to see if we can just change name set by the FLUENTD code
			fluentdVolMount.Name = "fluentd-config"
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

	return nil
}

// addJvmArgs adds the additional JVM args needed to enable and configure logging
// in the Coherence container
func addJvmArgs(coherenceSpec map[string]interface{}) {
	var jvm map[string]interface{}
	if val, found := coherenceSpec["jvm"]; !found {
		jvm = make(map[string]interface{})
		coherenceSpec["jvm"] = jvm
	} else {
		jvm = val.(map[string]interface{})
	}

	var args []interface{}
	if val, found := jvm["args"]; !found {
		args = additionalJvmArgs
	} else {
		// just append our logging args, this needs to be improved to handle
		// the case where one or more of the args are already present
		args = val.([]interface{})
		args = append(args, additionalJvmArgs...)
	}
	jvm["args"] = args
}

// handleDelete handles delete processing - we delete the Coherence CR when our VerrazzanoCoherenceWorkload is deleted.
// returns true if our workload is being deleted, false otherwise
func (r *Reconciler) handleDelete(ctx context.Context, log logr.Logger, workload *vzapi.VerrazzanoCoherenceWorkload, coherence *unstructured.Unstructured) (bool, error) {
	if !workload.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllers.StringSliceContainsString(workload.ObjectMeta.Finalizers, finalizer) {
			log.Info("Deleting Coherence CR")
			if err := r.Delete(ctx, coherence); err != nil {
				return true, err
			}

			workload.ObjectMeta.Finalizers = controllers.RemoveStringFromStringSlice(workload.ObjectMeta.Finalizers, finalizer)
			if err := r.Update(ctx, workload); err != nil {
				// just log and keep going
				r.Log.Info("Unable to remove finalizer from workload", "error", err)
			}
		}

		return true, nil
	}

	// not deleting, so add our finalizer and update the workload if it's not already in the list
	if !controllers.StringSliceContainsString(workload.ObjectMeta.Finalizers, finalizer) {
		workload.ObjectMeta.Finalizers = append(workload.ObjectMeta.Finalizers, finalizer)
		if err := r.Update(ctx, workload); err != nil {
			// just log and keep going
			r.Log.Info("Unable to add finalizer to workload", "error", err)
		}
	}

	return false, nil
}
