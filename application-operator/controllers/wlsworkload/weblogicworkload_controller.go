// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsworkload

import (
	"context"
	"errors"

	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/loggingscope"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const specField = "spec"

var specServerPodFields = []string{specField, "serverPod"}
var specServerPodLabelsFields = append(specServerPodFields, "labels")
var specServerPodContainersFields = append(specServerPodFields, "containers")
var specServerPodVolumesFields = append(specServerPodFields, "volumes")
var specServerPodVolumeMountsFields = append(specServerPodFields, "volumeMounts")

// this struct allows us to extract information from the unstructured WebLogic spec
// so we can interface with the FLUENTD code
type containersMountsVolumes struct {
	Containers   []corev1.Container
	Volumes      []corev1.Volume
	VolumeMounts []corev1.VolumeMount
}

// Reconciler reconciles a VerrazzanoWebLogicWorkload object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vzapi.VerrazzanoWebLogicWorkload{}).
		Complete(r)
}

// Reconcile reconciles a VerrazzanoWebLogicWorkload resource. It fetches the embedded WebLogic CR, mutates it to add
// scopes and traits, and then writes out the CR (or deletes it if the workload is being deleted).
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanoweblogicworkloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanoweblogicworkloads/status,verbs=get;update;patch
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("verrazzanoweblogicworkload", req.NamespacedName)
	log.Info("Reconciling verrazzano weblogic workload")

	// fetch the workload and unwrap the WebLogic resource
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
		return reconcile.Result{}, errors.New("unable to determine contained GroupVersionKind for workload")
	}

	apiVersion, kind := gvk.ToAPIVersionAndKind()
	u.SetAPIVersion(apiVersion)
	u.SetKind(kind)

	// mutate the WebLogic domain resource, copy labels, add logging, etc.
	if err = copyLabels(log, workload.ObjectMeta.GetLabels(), u); err != nil {
		return reconcile.Result{}, err
	}

	if err = r.addLogging(ctx, log, req.NamespacedName.Namespace, workload.ObjectMeta.Labels, u); err != nil {
		return reconcile.Result{}, err
	}

	// set controller reference so the WebLogic domain CR gets deleted when the workload is deleted
	if err = controllerutil.SetControllerReference(workload, u, r.Scheme); err != nil {
		log.Error(err, "Unable to set controller ref")
		return reconcile.Result{}, err
	}

	// write out the WebLogic domain resource
	if err = r.Client.Create(ctx, u); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return reconcile.Result{}, err
		}
		log.Info("WebLogic domain CR already exists, ignoring error on create")
		return reconcile.Result{}, nil
	}

	log.Info("Successfully created WebLogic domain")
	return reconcile.Result{}, nil
}

// fetchWorkload fetches the VerrazzanoWebLogicWorkload data given a namespaced name
func (r *Reconciler) fetchWorkload(ctx context.Context, name types.NamespacedName) (*vzapi.VerrazzanoWebLogicWorkload, error) {
	var workload vzapi.VerrazzanoWebLogicWorkload
	if err := r.Get(ctx, name, &workload); err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Info("VerrazzanoWebLogicWorkload has been deleted", "name", name)
		} else {
			r.Log.Error(err, "Failed to fetch VerrazzanoWebLogicWorkload", "name", name)
		}
		return nil, err
	}

	return &workload, nil
}

// copyLabels copies specific labels from the Verrazzano workload to the contained WebLogic resource
func copyLabels(log logr.Logger, workloadLabels map[string]string, weblogic *unstructured.Unstructured) error {
	// the WebLogic domain spec/serverPod/labels field has labels that get propagated to the pods
	labels, found, _ := unstructured.NestedStringMap(weblogic.Object, specServerPodLabelsFields...)
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

	err := unstructured.SetNestedStringMap(weblogic.Object, labels, specServerPodLabelsFields...)
	if err != nil {
		log.Error(err, "Unable to set labels in spec serverPod")
		return err
	}

	return nil
}

// addLogging adds a FLUENTD sidecar and updates the WebLogic spec if there is an associated LoggingScope
func (r *Reconciler) addLogging(ctx context.Context, log logr.Logger, namespace string, labels map[string]string, weblogic *unstructured.Unstructured) error {
	loggingScope, err := vznav.LoggingScopeFromWorkloadLabels(ctx, r.Client, namespace, labels)
	if err != nil {
		return err
	}

	if loggingScope == nil {
		log.Info("No logging scope found for workload, nothing to do")
		return nil
	}

	// extract just enough of the WebLogic data into concrete types so we can merge with
	// the FLUENTD data
	var extracted containersMountsVolumes
	if serverPod, found, _ := unstructured.NestedMap(weblogic.Object, specServerPodFields...); found {
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(serverPod, &extracted); err != nil {
			return errors.New("unable to extract containers, volumes, and volume mounts from WebLogic spec")
		}
	}

	name, found, _ := unstructured.NestedString(weblogic.Object, "metadata", "name")
	if !found {
		return errors.New("expected to find metadata name in WebLogic spec")
	}

	// fluentdPod starts with what's in the spec and we add in the FLUENTD things when Apply is
	// called on the fluentdManager
	fluentdPod := &loggingscope.FluentdPod{
		Containers:   extracted.Containers,
		Volumes:      extracted.Volumes,
		VolumeMounts: extracted.VolumeMounts,
		LogPath:      loggingscope.BuildWLSLogPath(name),
		HandlerEnv:   loggingscope.GetWlsSpecificContainerEnv(),
	}
	fluentdManager := loggingscope.GetFluentd(ctx, r.Log, r.Client)

	// fluentdManager.Apply wants a QRR but it only cares about the namespace (at least for
	// this use case)
	resource := vzapi.QualifiedResourceRelation{Namespace: namespace}

	// note that this call has the side effect of creating a FLUENTD config map if one
	// does not already exist in the namespace
	if _, err = fluentdManager.Apply(loggingScope, resource, fluentdPod); err != nil {
		return err
	}

	// convert the containers, volumes, and mounts in fluentdPod to unstructured and set
	// the values in the spec
	fluentdPodUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(fluentdPod)
	if err != nil {
		return err
	}

	err = unstructured.SetNestedSlice(weblogic.Object, fluentdPodUnstructured["containers"].([]interface{}), specServerPodContainersFields...)
	if err != nil {
		log.Error(err, "Unable to set serverPod containers")
		return err
	}
	err = unstructured.SetNestedSlice(weblogic.Object, fluentdPodUnstructured["volumes"].([]interface{}), specServerPodVolumesFields...)
	if err != nil {
		log.Error(err, "Unable to set serverPod volumes")
		return err
	}
	err = unstructured.SetNestedField(weblogic.Object, fluentdPodUnstructured["volumeMounts"].([]interface{}), specServerPodVolumeMountsFields...)
	if err != nil {
		log.Error(err, "Unable to set serverPod volumeMounts")
		return err
	}

	// logHome and logHomeEnabled fields need to be set to turn on logging
	err = unstructured.SetNestedField(weblogic.Object, loggingscope.BuildWLSLogHome(name), specField, "logHome")
	if err != nil {
		log.Error(err, "Unable to set logHome")
		return err
	}
	err = unstructured.SetNestedField(weblogic.Object, true, specField, "logHomeEnabled")
	if err != nil {
		log.Error(err, "Unable to set logHomeEnabled")
		return err
	}

	return nil
}
