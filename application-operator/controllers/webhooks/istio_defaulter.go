// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gertd/go-pluralize"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"go.uber.org/zap"
	securityv1beta1 "istio.io/api/security/v1beta1"
	"istio.io/api/type/v1beta1"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	istioversionedclient "istio.io/client-go/pkg/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// IstioDefaulterPath specifies the path of Istio defaulter webhook
const IstioDefaulterPath = "/istio-defaulter"

// IstioAppLabel label to be used for all pods that are istio enabled
const IstioAppLabel = "verrazzano.io/istio"

// IstioWebhook type for istio defaulter webhook
type IstioWebhook struct {
	client.Client
	IstioClient   istioversionedclient.Interface
	Decoder       *admission.Decoder
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface
}

// Handle is the entry point for the mutating webhook.
// This function is called for any pods that are created in a namespace with the label istio-injection=enabled.
func (a *IstioWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	counterMetricObject, errorCounterMetricObject, handleDurationMetricObject, zapLogForMetrics, err := metricsexporter.ExposeControllerMetrics("IstioDefaulter", metricsexporter.IstioHandleCounter, metricsexporter.IstioHandleError, metricsexporter.IstioHandleDuration)
	if err != nil {
		return admission.Response{}
	}
	handleDurationMetricObject.TimerStart()
	defer handleDurationMetricObject.TimerStop()

	var log = zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldWebhook, "istio-defaulter")

	pod := &corev1.Pod{}
	err = a.Decoder.Decode(req, pod)
	if err != nil {
		errorCounterMetricObject.Inc(zapLogForMetrics, err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Check for the annotation "sidecar.istio.io/inject: false".  No action required if annotation is set to false.
	for key, value := range pod.Annotations {
		if key == "sidecar.istio.io/inject" && value == "false" {
			log.Debugf("Pod labeled with sidecar.istio.io/inject: false: %s:%s:%s", req.Namespace, pod.Name, pod.GenerateName)
			return admission.Allowed("No action required, pod labeled with sidecar.istio.io/inject: false")
		}
	}

	// Get all owner references for this pod
	ownerRefList, err := a.flattenOwnerReferences(nil, req.Namespace, pod.OwnerReferences, log)
	if err != nil {
		errorCounterMetricObject.Inc(zapLogForMetrics, err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Check if the pod was created from an ApplicationConfiguration resource.
	// We do this by checking for the existence of an ApplicationConfiguration ownerReference resource.
	appConfigOwnerRef := metav1.OwnerReference{}
	for _, ownerRef := range ownerRefList {
		if ownerRef.Kind == "ApplicationConfiguration" {
			appConfigOwnerRef = ownerRef
			break
		}
	}
	// No ApplicationConfiguration ownerReference resource was found so there is no action required.
	if appConfigOwnerRef == (metav1.OwnerReference{}) {
		log.Debugf("Pod is not a child of an ApplicationConfiguration: %s:%s:%s", req.Namespace, pod.Name, pod.GenerateName)
		return admission.Allowed("No action required, pod is not a child of an ApplicationConfiguration resource")
	}

	// If a pod is using the "default" service account then create a app specific service account, if not already
	// created.  A service account is used as a principal in the Istio Authorization policy we create/update.
	serviceAccountName := pod.Spec.ServiceAccountName
	if serviceAccountName == "default" || serviceAccountName == "" {
		serviceAccountName, err = a.createServiceAccount(req.Namespace, appConfigOwnerRef, log)
		if err != nil {
			errorCounterMetricObject.Inc(zapLogForMetrics, err)
			return admission.Errored(http.StatusInternalServerError, err)
		}
	}

	// Create/update Istio Authorization policy for the given pod.
	err = a.createUpdateAuthorizationPolicy(req.Namespace, serviceAccountName, appConfigOwnerRef, pod.ObjectMeta.Labels, log)
	if err != nil {
		errorCounterMetricObject.Inc(zapLogForMetrics, err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Fixup Istio Authorization policies within a project
	ap := &AuthorizationPolicy{
		Client:      a.Client,
		IstioClient: a.IstioClient,
	}
	err = ap.fixupAuthorizationPoliciesForProjects(req.Namespace, log)
	if err != nil {
		errorCounterMetricObject.Inc(zapLogForMetrics, err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Add the label to the pod which is used as the match selector in the authorization policy we created/updated.
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[IstioAppLabel] = appConfigOwnerRef.Name

	// Set the service account name for the pod which is used in the principal portion of the authorization policy we
	// created/updated.
	pod.Spec.ServiceAccountName = serviceAccountName

	// Marshal the mutated pod to send back in the admission review response.
	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		errorCounterMetricObject.Inc(zapLogForMetrics, err)
		return admission.Errored(http.StatusInternalServerError, err)
	}
	counterMetricObject.Inc(zapLogForMetrics, err)
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// InjectDecoder injects the decoder.
func (a *IstioWebhook) InjectDecoder(d *admission.Decoder) error {
	a.Decoder = d
	return nil
}

// createUpdateAuthorizationPolicy will create/update an Istio authoriztion policy.
func (a *IstioWebhook) createUpdateAuthorizationPolicy(namespace string, serviceAccountName string, ownerRef metav1.OwnerReference, labels map[string]string, log *zap.SugaredLogger) error {
	podPrincipal := fmt.Sprintf("cluster.local/ns/%s/sa/%s", namespace, serviceAccountName)
	gwPrincipal := "cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account"
	promPrincipal := "cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-wls"
	weblogicOperPrincipal := "cluster.local/ns/verrazzano-system/sa/weblogic-wls-sa"
	promOperatorPrincipal := "cluster.local/ns/verrazzano-monitoring/sa/prometheus-wls-kube-p-prometheus"

	principals := []string{
		podPrincipal,
		gwPrincipal,
		promPrincipal,
		promOperatorPrincipal,
	}
	// If the pod is WebLogic then add the WebLogic wls principle so that the wls can
	// communicate with the WebLogic servers
	workloadType, found := labels[constants.LabelWorkloadType]
	weblogicFound := found && workloadType == constants.WorkloadTypeWeblogic
	if weblogicFound {
		principals = append(principals, weblogicOperPrincipal)
	}

	// Check if authorization policy exist.  The name of the authorization policy is the owner reference name which happens
	// to be the appconfig name.
	authPolicy, err := a.IstioClient.SecurityV1beta1().AuthorizationPolicies(namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})

	// If the authorization policy does not exist then we create it.
	if err != nil && errors.IsNotFound(err) {
		selector := v1beta1.WorkloadSelector{
			MatchLabels: map[string]string{
				IstioAppLabel: ownerRef.Name,
			},
		}
		fromRules := []*securityv1beta1.Rule_From{
			{
				Source: &securityv1beta1.Source{
					Principals: principals,
				},
			},
		}

		ap := &clisecurity.AuthorizationPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ownerRef.Name,
				Namespace: namespace,
				Labels: map[string]string{
					IstioAppLabel: ownerRef.Name,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						Name:       ownerRef.Name,
						Kind:       ownerRef.Kind,
						APIVersion: ownerRef.APIVersion,
						UID:        ownerRef.UID,
					},
				},
			},
			Spec: securityv1beta1.AuthorizationPolicy{
				Selector: &selector,
				Rules: []*securityv1beta1.Rule{
					{
						From: fromRules,
					},
				},
			},
		}

		log.Infof("Creating Istio authorization policy: %s:%s", namespace, ownerRef.Name)
		_, err := a.IstioClient.SecurityV1beta1().AuthorizationPolicies(namespace).Create(context.TODO(), ap, metav1.CreateOptions{})
		return err
	} else if err != nil {
		return err
	}

	// If the pod and/or WebLogic wls principals are missing then update the principal list
	principalSet := vzstring.SliceToSet(authPolicy.Spec.GetRules()[0].From[0].Source.Principals)
	var update bool
	if _, ok := principalSet[podPrincipal]; !ok {
		update = true
		authPolicy.Spec.GetRules()[0].From[0].Source.Principals = append(authPolicy.Spec.GetRules()[0].From[0].Source.Principals, podPrincipal)
	}
	if weblogicFound {
		if _, ok := principalSet[weblogicOperPrincipal]; !ok {
			update = true
			authPolicy.Spec.GetRules()[0].From[0].Source.Principals = append(authPolicy.Spec.GetRules()[0].From[0].Source.Principals, weblogicOperPrincipal)
		}
	}
	// Update the policy with the principals that are missing
	if update {
		log.Debugf("Updating Istio authorization policy: %s:%s", namespace, ownerRef.Name)
		_, err := a.IstioClient.SecurityV1beta1().AuthorizationPolicies(namespace).Update(context.TODO(), authPolicy, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

// createServiceAccount will create a service account to be referenced by the Istio authorization policy
func (a *IstioWebhook) createServiceAccount(namespace string, ownerRef metav1.OwnerReference, log *zap.SugaredLogger) (string, error) {
	// Check if service account exist.  The name of the service account is the owner reference name which happens
	// to be the appconfig name.
	serviceAccount, err := a.KubeClient.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})

	// If the service account does not exist then we create it.
	if err != nil && errors.IsNotFound(err) {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ownerRef.Name,
				Namespace: namespace,
				Labels: map[string]string{
					IstioAppLabel: ownerRef.Name,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						Name:       ownerRef.Name,
						Kind:       ownerRef.Kind,
						APIVersion: ownerRef.APIVersion,
						UID:        ownerRef.UID,
					},
				},
			},
		}
		log.Debugf("Creating service account: %s:%s", namespace, ownerRef.Name)
		serviceAccount, err = a.KubeClient.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), sa, metav1.CreateOptions{})
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	return serviceAccount.Name, nil
}

// flattenOwnerReferences traverses a nested array of owner references and returns a single array of owner references.
func (a *IstioWebhook) flattenOwnerReferences(list []metav1.OwnerReference, namespace string, ownerRefs []metav1.OwnerReference, log *zap.SugaredLogger) ([]metav1.OwnerReference, error) {
	for _, ownerRef := range ownerRefs {
		list = append(list, ownerRef)

		group, version := controllers.ConvertAPIVersionToGroupAndVersion(ownerRef.APIVersion)
		resource := schema.GroupVersionResource{
			Group:    group,
			Version:  version,
			Resource: pluralize.NewClient().Plural(strings.ToLower(ownerRef.Kind)),
		}

		unst, err := a.DynamicClient.Resource(resource).Namespace(namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed getting the Dynamic API: %v", err)
			}
			return nil, err
		}

		if len(unst.GetOwnerReferences()) != 0 {
			list, err = a.flattenOwnerReferences(list, namespace, unst.GetOwnerReferences(), log)
			if err != nil {
				return nil, err
			}
		}
	}
	return list, nil
}
