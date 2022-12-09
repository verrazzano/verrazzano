// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gertd/go-pluralize"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
)

// MySQLStatefulSetWebhook type for Verrazzano mysql backup webhook
type MySQLStatefulSetWebhook struct {
	client.Client
	//IstioClient   istioversionedclient.Interface
	Decoder       *admission.Decoder
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface
	Defaulters    []MySQLDefaulter
}

// Handle is the entry point for the mutating webhook.
// This function is called for any jobs that are created in a namespace with the label istio-injection=enabled.
func (m *MySQLStatefulSetWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {

	var log = zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldWebhook, "mysql-statefulset")

	sts := &appsv1.StatefulSet{}
	err := m.Decoder.Decode(req, sts)
	if err != nil {
		log.Error("Unable to decode statefulset due to ", zap.Error(err))
		return admission.Errored(http.StatusBadRequest, err)
	}
	return m.processStatefulSet(req, sts, log)
}

// InjectDecoder injects the decoder.
func (m *MySQLStatefulSetWebhook) InjectDecoder(d *admission.Decoder) error {
	m.Decoder = d
	return nil
}

// processStatefulSet processes the statefulset request and applies the necessary annotations based on Job ownership and labels
func (m *MySQLStatefulSetWebhook) processStatefulSet(req admission.Request, sts *appsv1.StatefulSet, log *zap.SugaredLogger) admission.Response {
	var mysqlOperatorOwnerReferencePresent, mysqlOperatorLabelPresent bool

	// Check for the annotation or label "sidecar.istio.io/inject: false".  No action required if it is set to false.
	for key, value := range sts.Spec.Template.Annotations {
		if key == "sidecar.istio.io/inject" && value == "false" {
			log.Debugf("StatefulSet is annotated with sidecar.istio.io/inject: false: %s:%s", sts.Namespace, sts.Name)
			return admission.Allowed("No action required, StatefulSet annotated with sidecar.istio.io/inject: false")
		}
	}
	for key, value := range sts.Spec.Template.Labels {
		if key == "sidecar.istio.io/inject" && value == "false" {
			log.Debugf("StatefulSet is labeled with sidecar.istio.io/inject: false: %s:%s", sts.Namespace, sts.Name)
			return admission.Allowed("No action required, StatefulSet labeled with sidecar.istio.io/inject: false")
		}
	}

	// Job spec annotation is only done for jobs launched by mysql operator
	for key, value := range sts.Labels {
		if key == MySQLOperatorLabel && value == MySQLOperatorJobLabelValue {
			mysqlOperatorLabelPresent = true
			break
		}
	}

	// Get all owner references for this sts
	ownerRefList, err := m.getSimplifiedOwnerReferences(nil, req.Namespace, sts.OwnerReferences, log)
	if err != nil {
		fmt.Println(err)
		log.Error("Unable to get owwner list due to ", zap.Error(err))
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Check if the statefulset was created by the InnoDBCluster
	for _, ownerRef := range ownerRefList {
		if ownerRef.Kind == "InnoDBCluster" && ownerRef.Name == "mysql" {
			mysqlOperatorOwnerReferencePresent = true
			break
		}
	}

	if !mysqlOperatorOwnerReferencePresent && !mysqlOperatorLabelPresent {
		log.Debugf("No annotation is required for this sts: %s:%s:%s", req.Namespace, sts.Name, sts.GenerateName)
		return admission.Allowed("No action required, sts not labelled with app.kubernetes.io/created-by: mysql-operator")
	}

	istioAnnotation := make(map[string]string)
	istioAnnotation[MySQLOperatorJobPodSpecAnnotationKey] = MySQLOperatorJobPodSpecAnnotationValue
	sts.Spec.Template.SetAnnotations(istioAnnotation)

	// Marshal the mutated pod to send back in the admission review response.
	marshaledStatefulSetData, err := json.Marshal(sts)
	if err != nil {
		log.Error("Unable to marshall data due to ", zap.Error(err))
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledStatefulSetData)

}

// getSimplifiedOwnerReferences traverses a nested array of owner references and returns a single array of owner references.
func (m *MySQLStatefulSetWebhook) getSimplifiedOwnerReferences(list []metav1.OwnerReference, namespace string, ownerRefs []metav1.OwnerReference, log *zap.SugaredLogger) ([]metav1.OwnerReference, error) {

	for _, ownerRef := range ownerRefs {
		list = append(list, ownerRef)

		group, version := ConvertAPIVersionToGroupAndVersion(ownerRef.APIVersion)
		resource := schema.GroupVersionResource{
			Group:    group,
			Version:  version,
			Resource: pluralize.NewClient().Plural(strings.ToLower(ownerRef.Kind)),
		}

		unst, err := m.DynamicClient.Resource(resource).Namespace(namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed getting the Dynamic API: %v", err)
			}
			return nil, err
		}

		if len(unst.GetOwnerReferences()) != 0 {
			list, err = m.getSimplifiedOwnerReferences(list, namespace, unst.GetOwnerReferences(), log)
			if err != nil {
				return nil, err
			}
		}
	}
	return list, nil
}
