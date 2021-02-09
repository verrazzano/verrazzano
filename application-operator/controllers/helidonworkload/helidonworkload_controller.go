// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonworkload

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"

	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vzapi.VerrazzanoHelidonWorkload{}).
		Complete(r)
}

// Reconcile reconciles a VerrazzanoHelidonWorkload resource. It fetches the embedded DeploymentSpec, mutates it to add
// scopes and traits, and then writes out the apps/Deployment (or deletes it if the workload is being deleted).
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanohelidonworkloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanohelidonworkloads/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("verrazzanohelidonworkload", req.NamespacedName)
	log.Info("Reconciling verrazzano helidon workload")

	// fetch the workload
	var workload vzapi.VerrazzanoHelidonWorkload
	if err := r.Get(ctx, req.NamespacedName, &workload); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("VerrazzanoHelidonWorkload has been deleted", "name", req.NamespacedName)
		} else {
			log.Error(err, "Failed to fetch VerrazzanoHelidonWorkload", "name", req.NamespacedName)
		}
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Got the workload", "apiVersion", workload.APIVersion, "kind", workload.Kind)

	//TODO: Find the resource object to record the event to, default is the parent appConfig - done in OAM

	//unwrap the apps/DeploymentSpec and meta/ObjectMeta
	deploy, err := r.convertWorkloadToDeployment(&workload)
	if err != nil {
		log.Error(err, "Failed to convert workload to deployment")
		//TODO: OAM is doing Wait
		return reconcile.Result{}, err
	}
	//TODO: make sure the namespace is set to the namespace of the component - was needed in coherence
	//TODO: set the controller reference so that we can watch this deployment and it will be deleted automatically
	/*if err := ctrl.SetControllerReference(&workload, deploy, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}*/

	// server side apply, only the fields we set are touched
	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner(workload.GetUID())}
	if err := r.Patch(ctx, deploy, client.Apply, applyOpts...); err != nil {
		log.Error(err, "Failed to apply a deployment")
		//TODO: OAM is doing Wait
		return reconcile.Result{}, err
	}

	// create a service for the workload
	service, err := r.createServiceFromDeployment(&workload, deploy)
	if err != nil {
		log.Error(err, "Failed to get service from a deployment")
		//TODO: OAM is doing Wait
		return reconcile.Result{}, err
	}
	//TODO: always set the controller reference so that we can watch this service and
	/*if err := ctrl.SetControllerReference(&workload, service, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}*/

	// server side apply the service
	if err := r.Patch(ctx, service, client.Apply, applyOpts...); err != nil {
		log.Error(err, "Failed to apply a service")
		//TODO: OAM is doing Wait
		return reconcile.Result{}, err
	}

	log.Info("Successfully created Verrazzano Helidon workload")
	return ctrl.Result{}, nil
}

//convertWorkloadToDeployment converts a VerrazzanoHelidonWorkload into a Deployment.
func (r *Reconciler) convertWorkloadToDeployment(
	workload *vzapi.VerrazzanoHelidonWorkload) (*appsv1.Deployment, error) {
	//TODO: What if metadata and spec are not set?
	d := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       deploymentKind,
			APIVersion: deploymentAPIVersion,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					labelKey: string(workload.GetUID()),
				},
			},
		},
	}
	workload.Spec.DeploymentTemplate.Metadata.DeepCopyInto(&d.ObjectMeta)
	workload.Spec.DeploymentTemplate.PodSpec.DeepCopyInto(&d.Spec.Template.Spec)
	//TODO: Set GUID from workload for controller reference work?
	d.Spec.Template.ObjectMeta.SetLabels(map[string]string{
		labelKey: string(workload.GetUID()),
	})
	//k8s server-side patch complains if the protocol is not set
	/*for i := 0; i < len(d.Spec.Template.Spec.Containers); i++ {
		for j := 0; j < len(d.Spec.Template.Spec.Containers[i].Ports); j++ {
			if len(d.Spec.Template.Spec.Containers[i].Ports[j].Protocol) == 0 {
				d.Spec.Template.Spec.Containers[i].Ports[j].Protocol = corev1.ProtocolTCP
			}
		}
	}*/
	//TODO: pass through label and annotation from the workload to the deployment - done in OAM
	//TODO: pass through label from the workload to the pod template - done in OAM
	if y, err := yaml.Marshal(d); err != nil {
		r.Log.Error(err, "Failed to convert deployment to yaml")
		r.Log.Info("Set deployment in json ", "DeploymentJson", d)
	} else {
		r.Log.Info("Set deployment in yaml ", "DeploymentYaml", string(y))
	}
	return d, nil
}

// create a service for the deployment
func (r *Reconciler) createServiceFromDeployment(workload *vzapi.VerrazzanoHelidonWorkload,
	deploy *appsv1.Deployment) (*corev1.Service, error) {

	// We don't add a Service if there are no containers for the Deployment.
	// This should never happen in practice.
	if len(deploy.Spec.Template.Spec.Containers) >= 1 {
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

		// We only add a single Service for the Deployment, even if multiple
		// ports or no ports are defined on the first container. This is to
		// exclude the need for implementing garbage collection in the
		// short-term in the case that ports are modified after creation.
		if len(deploy.Spec.Template.Spec.Containers[0].Ports) > 0 {
			s.Spec.Ports = []corev1.ServicePort{
				{
					Name:       deploy.GetName(),
					Port:       deploy.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort,
					TargetPort: intstr.FromInt(int(deploy.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)),
					Protocol:   corev1.ProtocolTCP,
				},
			}
		}
		if y, err := yaml.Marshal(s); err != nil {
			r.Log.Error(err, "Failed to convert service to yaml")
			r.Log.Info("Set service in json ", "deploy", s)
		} else {
			r.Log.Info("Set service in yaml: ", string(y))
		}
		return s, nil
	}
	return nil, nil
}
