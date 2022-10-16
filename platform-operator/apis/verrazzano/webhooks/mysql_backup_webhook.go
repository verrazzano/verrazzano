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
	batchv1 "k8s.io/api/batch/v1"
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

const (

	// MySQLOperatorJobLabel and MySQLOperatorJobLabelValue are the label key,value pair
	// that are applied to the job by mysql-operator.

	MySQLOperatorJobLabel      = "app.kubernetes.io/created-by"
	MySQLOperatorJobLabelValue = "mysql-operator"

	// MySQLOperatorJobPodSpecAnnotationKey and MySQLOperatorJobPodSpecAnnotationValue are
	// applied to the job spec so that it can talk to the k8s api server.

	MySQLOperatorJobPodSpecAnnotationKey   = "traffic.sidecar.istio.io/excludeOutboundPorts"
	MySQLOperatorJobPodSpecAnnotationValue = "443"
)

// MySQLBackupJobWebhook type for Verrazzano mysql backup webhook
type MySQLBackupJobWebhook struct {
	client.Client
	//IstioClient   istioversionedclient.Interface
	Decoder       *admission.Decoder
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface
	Defaulters    []MySQLDefaulter
}
type MySQLDefaulter interface {
}

// ConvertAPIVersionToGroupAndVersion splits APIVersion into API and version parts.
// An APIVersion takes the form api/version (e.g. networking.k8s.io/v1)
// If the input does not contain a / the group is defaulted to the empty string.
// apiVersion - The combined api and version to split
func ConvertAPIVersionToGroupAndVersion(apiVersion string) (string, string) {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) < 2 {
		// Use empty group for core types.
		return "", parts[0]
	}
	return parts[0], parts[1]
}

// Handle is the entry point for the mutating webhook.
// This function is called for any jobs that are created in a namespace with the label istio-injection=enabled.
func (m *MySQLBackupJobWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {

	var log = zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldWebhook, "mysql-backup")

	job := &batchv1.Job{}
	err := m.Decoder.Decode(req, job)
	if err != nil {
		log.Error("Unable to decode job due to ", zap.Error(err))
		return admission.Errored(http.StatusBadRequest, err)
	}
	return m.processJob(req, job, log)
}

// InjectDecoder injects the decoder.
func (m *MySQLBackupJobWebhook) InjectDecoder(d *admission.Decoder) error {
	m.Decoder = d
	return nil
}

// processJob processes the job request and applies the necessary annotations based on Job ownership and labels
func (m *MySQLBackupJobWebhook) processJob(req admission.Request, job *batchv1.Job, log *zap.SugaredLogger) admission.Response {
	var mysqlOperatorOwnerReferencePresent, mysqlOperatorLabelPresent bool

	// Check for the annotation "sidecar.istio.io/inject: false".  No action required if annotation is set to false.
	for key, value := range job.Annotations {
		if key == "sidecar.istio.io/inject" && value == "false" {
			log.Debugf("Job labeled with sidecar.istio.io/inject: false: %s:%s:%s", job.Namespace, job.Name, job.GenerateName)
			return admission.Allowed("No action required, job labeled with sidecar.istio.io/inject: false")
		}
	}

	// Job spec annotation is only done for jobs launched by mysql operator
	for key, value := range job.Labels {
		if key == MySQLOperatorJobLabel && value == MySQLOperatorJobLabelValue {
			mysqlOperatorLabelPresent = true
			break
		}
	}

	// Get all owner references for this job
	ownerRefList, err := m.getSimplifiedOwnerReferences(nil, req.Namespace, job.OwnerReferences, log)
	if err != nil {
		fmt.Println(err)
		log.Error("Unable to get owwner list due to ", zap.Error(err))
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Check if the Job was created from an CronJob or MySQLBackup resource.
	// We do this by checking for the existence of an ApplicationConfiguration ownerReference resource.
	for _, ownerRef := range ownerRefList {
		// This condition is satisfied by any Job for backup created from mysql-operator
		if ownerRef.Kind == "MySQLBackup" {
			mysqlOperatorOwnerReferencePresent = true
			break
		}
		// This condition is satisfied by a Job that is created from a cron-job (backup schedule) that is created from mysql-operator
		if ownerRef.Kind == "CronJob" {
			ok, err := m.isCronJobCreatedByMysqlOperator(req, ownerRef, log)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
			// ok = true when the job is triggered from a cronjob that was created by mysql operator
			if ok {
				mysqlOperatorOwnerReferencePresent = true
				break
			}
		}
	}

	if !mysqlOperatorOwnerReferencePresent && !mysqlOperatorLabelPresent {
		log.Debugf("No annotation is required for this job: %s:%s:%s", req.Namespace, job.Name, job.GenerateName)
		return admission.Allowed("No action required, job not labelled with app.kubernetes.io/created-by: mysql-operator")
	}
	istioAnnotation := make(map[string]string)
	istioAnnotation[MySQLOperatorJobPodSpecAnnotationKey] = MySQLOperatorJobPodSpecAnnotationValue
	job.Spec.Template.SetAnnotations(istioAnnotation)

	// Marshal the mutated pod to send back in the admission review response.
	marshaledJobData, err := json.Marshal(job)
	if err != nil {
		log.Error("Unable to marshall data due to ", zap.Error(err))
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledJobData)

}

// getSimplifiedOwnerReferences traverses a nested array of owner references and returns a single array of owner references.
func (m *MySQLBackupJobWebhook) getSimplifiedOwnerReferences(list []metav1.OwnerReference, namespace string, ownerRefs []metav1.OwnerReference, log *zap.SugaredLogger) ([]metav1.OwnerReference, error) {

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

func (m *MySQLBackupJobWebhook) isCronJobCreatedByMysqlOperator(req admission.Request, ownerRef metav1.OwnerReference, log *zap.SugaredLogger) (bool, error) {

	var mysqlOperatorOwnerReferencePresent, mysqlOperatorLabelPresent bool

	cjob, err := m.KubeClient.BatchV1().CronJobs(req.Namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
	if err != nil {
		log.Debugf("Unable to fetch cronJob : %s:%s", cjob.Namespace, cjob.Name)
		return false, err
	}
	cjOwnerRefList, err := m.getSimplifiedOwnerReferences(nil, req.Namespace, cjob.OwnerReferences, log)
	if err != nil {
		log.Error("Unable to get owner list due to ", zap.Error(err))
		return false, err
	}
	for _, cjOwnerRef := range cjOwnerRefList {
		if cjOwnerRef.Kind == "InnoDBCluster" {
			mysqlOperatorOwnerReferencePresent = true
			break
		}
	}

	// CronJob spec annotation is only done for jobs launched by mysql operator
	for key, value := range cjob.Labels {
		if key == MySQLOperatorJobLabel && value == MySQLOperatorJobLabelValue {
			mysqlOperatorLabelPresent = true
			break
		}
	}

	if mysqlOperatorOwnerReferencePresent && mysqlOperatorLabelPresent {
		// This is a litmus test that the cronjob was created by mysql-operator
		return true, nil
	}

	return false, fmt.Errorf("Cronjob was not created by MySQL operator")
}
