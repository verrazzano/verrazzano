// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonworkload

import (
	"context"
	"errors"
	"fmt"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/verrazzano/verrazzano/application-operator/controllers/appconfig"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

const (
	labelKey = "verrazzanohelidonworkloads.oam.verrazzano.io"
)

var (
	deploymentKind       = reflect.TypeOf(appsv1.Deployment{}).Name()
	deploymentAPIVersion = appsv1.SchemeGroupVersion.String()
	serviceKind          = reflect.TypeOf(corev1.Service{}).Name()
	serviceAPIVersion    = corev1.SchemeGroupVersion.String()
)

// Reconciler reconciles a VerrazzanoHelidonWorkload object
type Reconciler struct {
	client.Client
	Log     logr.Logger
	Scheme  *runtime.Scheme
	Metrics *metricstrait.Reconciler
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vzapi.VerrazzanoHelidonWorkload{}).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.Service{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

// Reconcile reconciles a VerrazzanoHelidonWorkload resource. It fetches the embedded DeploymentSpec, mutates it to add
// scopes and traits, and then writes out the apps/Deployment (or deletes it if the workload is being deleted).
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("verrazzanohelidonworkload", req.NamespacedName)
	log.Info("Reconciling VerrazzanoHelidonWorkload")

	// fetch the workload
	var workload vzapi.VerrazzanoHelidonWorkload
	if err := r.Get(ctx, req.NamespacedName, &workload); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("VerrazzanoHelidonWorkload has been deleted")
		} else {
			log.Error(err, "Failed to fetch VerrazzanoHelidonWorkload")
		}
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Retrieved workload", "apiVersion", workload.APIVersion, "kind", workload.Kind)

	// if required info is not available in workload, log error and return
	if len(workload.Spec.DeploymentTemplate.Metadata.GetName()) == 0 {
		err := errors.New("VerrazzanoHelidonWorkload is missing required spec.deploymentTemplate.metadata.name")
		log.Error(err, "workload", workload)
		return reconcile.Result{Requeue: false}, err
	}

	// unwrap the apps/DeploymentSpec and meta/ObjectMeta
	deploy, err := r.convertWorkloadToDeployment(&workload)
	if err != nil {
		log.Error(err, "Failed to convert workload to deployment")
		return reconcile.Result{}, err
	}

	// Attempt to get the existing deployment. This is used in the case where we don't want to update any resources
	// which are defined by Verrazzano such as the Fluentd image used by logging. In this case we obtain the previous
	// Fluentd image and set that on the new deployment. We also need to know if the deployment exists
	// so that when we write out the deployment later, we will call update instead of create if the deployment exists.
	var existingDeployment appsv1.Deployment
	deploymentKey := types.NamespacedName{Name: workload.Spec.DeploymentTemplate.Metadata.GetName(), Namespace: workload.Namespace}
	if err := r.Get(ctx, deploymentKey, &existingDeployment); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("No existing deployment found")
		} else {
			log.Error(err, "An error occurred trying to obtain an existing deployment")
			return reconcile.Result{}, err
		}
	}

	if err = r.addMetrics(ctx, log, req.NamespacedName.Namespace, &workload, deploy); err != nil {
		return reconcile.Result{}, err
	}

	// set the controller reference so that we can watch this deployment and it will be deleted automatically
	if err := ctrl.SetControllerReference(&workload, deploy, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// server side apply, only the fields we set are touched
	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner(workload.GetUID())}
	if err := r.Patch(ctx, deploy, client.Apply, applyOpts...); err != nil {
		log.Error(err, "Failed to apply a deployment")
		return reconcile.Result{}, err
	}

	// create a service for the workload
	service, err := r.createServiceFromDeployment(&workload, deploy)
	if err != nil {
		log.Error(err, "Failed to get service from a deployment")
		return reconcile.Result{}, err
	}
	// set the controller reference so that we can watch this service and it will be deleted automatically
	if err := ctrl.SetControllerReference(&workload, service, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// server side apply the service
	if err := r.Patch(ctx, service, client.Apply, applyOpts...); err != nil {
		log.Error(err, "Failed to apply a service")
		return reconcile.Result{}, err
	}

	// write out restart-version in helidon deployment
	if err = r.restartHelidon(ctx, workload.Annotations[vzconst.RestartVersionAnnotation], &workload, log); err != nil {
		return reconcile.Result{}, err
	}

	// Prepare the list of resources to reference in status.
	statusResources := []vzapi.QualifiedResourceRelation{
		{
			APIVersion: deploy.GetObjectKind().GroupVersionKind().GroupVersion().String(),
			Kind:       deploy.GetObjectKind().GroupVersionKind().Kind,
			Name:       deploy.GetName(),
			Namespace:  deploy.GetNamespace(),
			Role:       "Deployment",
		},
		{
			APIVersion: service.GetObjectKind().GroupVersionKind().GroupVersion().String(),
			Kind:       service.GetObjectKind().GroupVersionKind().Kind,
			Name:       service.GetName(),
			Namespace:  service.GetNamespace(),
			Role:       "Service",
		},
	}

	if !vzapi.QualifiedResourceRelationSlicesEquivalent(statusResources, workload.Status.Resources) {
		workload.Status.Resources = statusResources
		if err := r.Status().Update(ctx, &workload); err != nil {
			return reconcile.Result{}, err
		}
	}

	log.Info("Successfully created Verrazzano Helidon workload")
	return reconcile.Result{}, nil
}

// convertWorkloadToDeployment converts a VerrazzanoHelidonWorkload into a Deployment.
func (r *Reconciler) convertWorkloadToDeployment(
	workload *vzapi.VerrazzanoHelidonWorkload) (*appsv1.Deployment, error) {

	d := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       deploymentKind,
			APIVersion: deploymentAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: workload.Spec.DeploymentTemplate.Metadata.GetName(),
			//make sure the namespace is set to the namespace of the component
			Namespace: workload.GetNamespace(),
		},
		Spec: appsv1.DeploymentSpec{
			//setting label selector for pod that this deployment will manage
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					labelKey: string(workload.GetUID()),
				},
			},
		},
	}
	// Set metadata on deployment from workload spec's metadata
	d.ObjectMeta.SetLabels(workload.Spec.DeploymentTemplate.Metadata.GetLabels())
	d.ObjectMeta.SetAnnotations(workload.Spec.DeploymentTemplate.Metadata.GetAnnotations())
	// Set deployment strategy from workload spec
	d.Spec.Strategy = workload.Spec.DeploymentTemplate.Strategy
	// Set PodSpec on deployment's PodTemplate from workload spec
	workload.Spec.DeploymentTemplate.PodSpec.DeepCopyInto(&d.Spec.Template.Spec)
	// making sure pods have same label as selector on deployment
	d.Spec.Template.ObjectMeta.SetLabels(map[string]string{
		labelKey: string(workload.GetUID()),
	})

	// pass through label and annotation from the workload to the deployment
	passLabelAndAnnotation(workload, d)

	if y, err := yaml.Marshal(d); err != nil {
		r.Log.Error(err, "Failed to convert deployment to yaml")
		r.Log.Info("Deployment in json format ", "DeploymentJson", d)
	} else {
		r.Log.V(1).Info("Deployment in yaml format ", "DeploymentYaml", string(y))
	}

	return d, nil
}

// createServiceFromDeployment creates a service for the deployment
func (r *Reconciler) createServiceFromDeployment(workload *vzapi.VerrazzanoHelidonWorkload,
	deploy *appsv1.Deployment) (*corev1.Service, error) {

	// We don't add a Service if there are no containers for the Deployment.
	// This should never happen in practice.
	if len(deploy.Spec.Template.Spec.Containers) > 0 {
		s := &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       serviceKind,
				APIVersion: serviceAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploy.GetName(),
				Namespace: deploy.GetNamespace(),
				Labels: map[string]string{
					labelKey: string(workload.GetUID()),
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: deploy.Spec.Selector.MatchLabels,
				Ports:    []corev1.ServicePort{},
				Type:     corev1.ServiceTypeClusterIP,
			},
		}

		for _, container := range deploy.Spec.Template.Spec.Containers {
			if len(container.Ports) > 0 {
				for _, port := range container.Ports {
					// All ports within a ServiceSpec must have unique names.
					// When considering the endpoints for a Service, this must match the 'name' field in the EndpointPort.
					name := strings.ToLower(string(corev1.ProtocolTCP)) + "-" + container.Name + "-" + strconv.FormatInt(int64(port.ContainerPort), 10)
					protocol := corev1.ProtocolTCP
					if len(port.Protocol) > 0 {
						protocol = port.Protocol
					}

					servicePort := corev1.ServicePort{
						Name:       name,
						Port:       port.ContainerPort,
						TargetPort: intstr.FromInt(int(port.ContainerPort)),
						Protocol:   protocol,
					}
					r.Log.V(1).Info("Appending port to service", "servicePort", servicePort)
					s.Spec.Ports = append(s.Spec.Ports, servicePort)
				}
			}
		}
		if y, err := yaml.Marshal(s); err != nil {
			r.Log.Error(err, "Failed to convert service to yaml")
			r.Log.Info("Service in json format ", "ServiceJson", s)
		} else {
			r.Log.V(1).Info("Service in yaml format: ", "ServiceYaml", string(y))
		}
		return s, nil
	}
	return nil, nil
}

// passLabelAndAnnotation passes through labels and annotation objectMeta from the workload to the deployment object
func passLabelAndAnnotation(workload *vzapi.VerrazzanoHelidonWorkload, deploy *appsv1.Deployment) {
	// set app-config labels on deployment metadata
	deploy.SetLabels(mergeMapOverrideWithDest(workload.GetLabels(), deploy.GetLabels()))
	// set app-config labels on deployment/podtemplate metadata
	deploy.Spec.Template.SetLabels(mergeMapOverrideWithDest(workload.GetLabels(), deploy.Spec.Template.GetLabels()))
	// set app-config annotation on deployment metadata
	deploy.SetAnnotations(mergeMapOverrideWithDest(workload.GetAnnotations(), deploy.GetAnnotations()))
}

// mergeMapOverrideWithDest merges two could be nil maps. If any conflicts, override src with dst.
func mergeMapOverrideWithDest(src, dst map[string]string) map[string]string {
	if src == nil && dst == nil {
		return nil
	}
	r := make(map[string]string)
	for k, v := range dst {
		r[k] = v
	}
	for k, v := range src {
		if _, exist := r[k]; !exist {
			r[k] = v
		}
	}
	return r
}

// addMetrics adds the labels and annotations needed for metrics to the Helidon resource annotations which are propagated to the individual Helidon pods.
func (r *Reconciler) addMetrics(ctx context.Context, log logr.Logger, namespace string, workload *vzapi.VerrazzanoHelidonWorkload, helidon *appsv1.Deployment) error {
	log.Info(fmt.Sprintf("Adding Metrics for workload: %s", workload.Name))
	metricsTrait, err := vznav.MetricsTraitFromWorkloadLabels(ctx, r.Client, log, namespace, workload.ObjectMeta)
	if err != nil {
		return err
	}

	if metricsTrait == nil {
		log.Info("Workload has no associated MetricTrait, nothing to do")
		return nil
	}
	log.Info(fmt.Sprintf("Found associated metrics trait for workload: %s : %s", workload.Name, metricsTrait.Name))

	traitDefaults, err := r.Metrics.NewTraitDefaultsForGenericWorkload()
	if err != nil {
		log.Error(err, "Unable to get default metric trait values")
		return err
	}

	if helidon.Spec.Template.Labels == nil {
		helidon.Spec.Template.Labels = make(map[string]string)
	}

	if helidon.Spec.Template.Annotations == nil {
		helidon.Spec.Template.Annotations = make(map[string]string)
	}

	labels := metricstrait.MutateLabels(metricsTrait, nil, helidon.Spec.Template.Labels)
	annotations := metricstrait.MutateAnnotations(metricsTrait, nil, traitDefaults, helidon.Spec.Template.Annotations)

	finalLabels := mergeMapOverrideWithDest(helidon.Spec.Template.Labels, labels)
	log.Info(fmt.Sprintf("Setting labels on %s: %v", workload.Name, finalLabels))
	helidon.Spec.Template.Labels = finalLabels
	finalAnnotations := mergeMapOverrideWithDest(helidon.Spec.Template.Annotations, annotations)
	log.Info(fmt.Sprintf("Setting annotations on %s: %v", workload.Name, finalAnnotations))
	helidon.Spec.Template.Annotations = finalAnnotations

	return nil
}

func (r *Reconciler) restartHelidon(ctx context.Context, restartVersion string, workload *vzapi.VerrazzanoHelidonWorkload, log logr.Logger) error {
	if len(restartVersion) > 0 {
		var deploymentList appsv1.DeploymentList
		componentNameReq, _ := labels.NewRequirement(oam.LabelAppComponent, selection.Equals, []string{workload.ObjectMeta.Labels[oam.LabelAppComponent]})
		appNameReq, _ := labels.NewRequirement(oam.LabelAppName, selection.Equals, []string{workload.ObjectMeta.Labels[oam.LabelAppName]})
		selector := labels.NewSelector()
		selector = selector.Add(*componentNameReq, *appNameReq)
		err := r.Client.List(ctx, &deploymentList, &client.ListOptions{Namespace: workload.Namespace, LabelSelector: selector})
		if err != nil {
			return err
		}
		for index := range deploymentList.Items {
			deployment := &deploymentList.Items[index]
			if err := appconfig.DoRestartDeployment(ctx, r.Client, restartVersion, deployment, log); err != nil {
				return err
			}
		}
	}
	return nil
}
