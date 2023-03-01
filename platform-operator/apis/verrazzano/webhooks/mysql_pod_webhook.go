// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"net/http"

	vzlog "github.com/verrazzano/verrazzano/pkg/log"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	mysqlServerLabelKey   = "app.kubernetes.io/name"
	mysqlServerLabelValue = "mysql-innodbcluster-mysql-server"
)

// MySQLPodWebhook type for Verrazzano mysql backup webhook
type MySQLPodWebhook struct {
	client.Client
	//IstioClient   istioversionedclient.Interface
	Decoder       *admission.Decoder
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface
	Defaulters    []MySQLDefaulter
}

// Handle is the entry point for the mutating webhook.
// This function is called for any pods that are created in a namespace with the label istio-injection=enabled.
func (m *MySQLPodWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {

	var log = zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldWebhook, "mysql-pod")

	pod := &corev1.Pod{}
	err := m.Decoder.Decode(req, pod)
	if err != nil {
		log.Error("Unable to decode pod due to ", zap.Error(err))
		return admission.Errored(http.StatusBadRequest, err)
	}
	return m.processPod(req, pod, log)
}

// InjectDecoder injects the decoder.
func (m *MySQLPodWebhook) InjectDecoder(d *admission.Decoder) error {
	m.Decoder = d
	return nil
}

// processPod processes the pod request and annotate if it is the appropriate pod
func (m *MySQLPodWebhook) processPod(req admission.Request, pod *corev1.Pod, log *zap.SugaredLogger) admission.Response {
	// Check to ensure this pod is the mysql server
	if pod.Labels[mysqlServerLabelKey] != mysqlServerLabelValue {
		return admission.Allowed("No action required, Pod is not the mysql server")
	}

	// Check to ensure this pod has been deployed by mysql-operator
	if pod.Labels[mySQLOperatorLabel] != mySQLOperatorLabelValue {
		log.Debugf("mysql pod does not have the correct mysql-operator label")
		return admission.Allowed("No action required, Pod has not been deployed by mysql-operator")
	}

	// Check for the annotation or label "sidecar.istio.io/inject: false".  No action required if it is set to false.
	if pod.Labels[sidecarIstioInjectKey] == sidecarIstioInjectValue || pod.Annotations[sidecarIstioInjectKey] == sidecarIstioInjectValue {
		log.Debugf("Pod is annotated or labled with sidecar.istio.io/inject: false: %s:%s", pod.Namespace, pod.Name)
		return admission.Allowed("No action required, Pod annotated or labeled with sidecar.istio.io/inject: false")
	}

	pod.SetAnnotations(map[string]string{mySQLIstioAnnotationKey: mySQLIstioAnnotationValue})

	// Marshal the mutated pod to send back in the admission review response.
	marshaledPodData, err := json.Marshal(pod)
	if err != nil {
		log.Error("Unable to marshall data due to ", zap.Error(err))
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPodData)

}
