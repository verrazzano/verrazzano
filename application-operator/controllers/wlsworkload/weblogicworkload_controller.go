// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsworkload

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	vzlogInit "github.com/verrazzano/verrazzano/pkg/log"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/controllers/logging"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
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
	"sigs.k8s.io/yaml"
)

const (
	metadataField                         = "metadata"
	specField                             = "spec"
	loggingNamePart                       = "logging-stdout"
	loggingMountPath                      = "/fluentd/etc/custom.conf"
	loggingKey                            = "custom.conf"
	defaultMode                     int32 = 400
	lastServerStartPolicyAnnotation       = "verrazzano-io/last-server-start-policy"
	Never                                 = "Never"
	NeverV8                               = "NEVER"
	IfNeeded                              = "IfNeeded"
	IfNeededV8                            = "IF_NEEDED"
	webLogicDomainUIDLabel                = "weblogic.domainUID"
	webLogicPluginConfigYamlKey           = "WebLogicPlugin.yaml"
	WDTConfigMapNameSuffix                = "-wdt-config-map"
	controllerName                        = "weblogicworkload"
	DomainKind                            = "Domain"
	ClusterKind                           = "Cluster"
	APIVersionV8                          = "weblogic.oracle/v8"
	APIVersionV9                          = "weblogic.oracle/v9"
	APIVersionV1                          = "weblogic.oracle/v1"
)

const defaultMonitoringExporterTemplate = `
  {
    {{.ImageSetting}}"imagePullPolicy": "IfNotPresent",
    "configuration": {
      "domainQualifier": true,
      "metricsNameSnakeCase": true,
      "queries": [
        {
           "key": "name",
           "keyName": "location",
           "prefix": "wls_server_",
           "applicationRuntimes": {
              "key": "name",
              "keyName": "app",
              "componentRuntimes": {
                 "prefix": "wls_webapp_config_",
                 "type": "WebAppComponentRuntime",
                 "key": "name",
                 "values": [
                    "deploymentState",
                    "contextRoot",
                    "sourceInfo",
                    "sessionsOpenedTotalCount",
                    "openSessionsCurrentCount",
                    "openSessionsHighCount"
                 ],
                 "servlets": {
                    "prefix": "wls_servlet_",
                    "key": "servletName"
                 }
              }
           }
        },
        {
           "JVMRuntime": {
              "prefix": "wls_jvm_",
              "key": "name"
           }
        },
        {
           "executeQueueRuntimes": {
              "prefix": "wls_socketmuxer_",
              "key": "name",
              "values": [
                 "pendingRequestCurrentCount"
              ]
           }
        },
        {
           "workManagerRuntimes": {
              "prefix": "wls_workmanager_",
              "key": "name",
              "values": [
                 "stuckThreadCount",
                 "pendingRequests",
                 "completedRequests"
              ]
           }
        },
        {
           "threadPoolRuntime": {
              "prefix": "wls_threadpool_",
              "key": "name",
              "values": [
                 "executeThreadTotalCount",
                 "queueLength",
                 "stuckThreadCount",
                 "hoggingThreadCount"
              ]
           }
        },
        {
           "JMSRuntime": {
              "key": "name",
              "keyName": "jmsruntime",
              "prefix": "wls_jmsruntime_",
              "JMSServers": {
                 "prefix": "wls_jms_",
                 "key": "name",
                 "keyName": "jmsserver",
                 "destinations": {
                    "prefix": "wls_jms_dest_",
                    "key": "name",
                    "keyName": "destination"
                 }
              }
           }
        },
        {
           "persistentStoreRuntimes": {
              "prefix": "wls_persistentstore_",
              "key": "name"
           }
        },
        {
           "JDBCServiceRuntime": {
              "JDBCDataSourceRuntimeMBeans": {
                 "prefix": "wls_datasource_",
                 "key": "name"
              }
           }
        },
        {
           "JTARuntime": {
              "prefix": "wls_jta_",
              "key": "name"
           }
        }
      ]
    }
  }
`

type defaultMonitoringExporterTemplateData struct {
	ImageSetting string
}

const defaultWDTConfigMapData = `
  {
    "resources": {
      "WebAppContainer": {
        "WeblogicPluginEnabled" : true
      }
    }
  }
`

var metaAnnotationFields = []string{metadataField, "annotations"}
var specDomainUID = []string{specField, "domainUID"}
var specServerPodFields = []string{specField, "serverPod"}
var specServerServiceFields = []string{specField, "serverService"}
var specServerPodLabelsFields = append(specServerPodFields, "labels")
var specServerPodContainersFields = append(specServerPodFields, "containers")
var specServerPodVolumesFields = append(specServerPodFields, "volumes")
var specServerPodVolumeMountsFields = append(specServerPodFields, "volumeMounts")
var specServerServiceLabelsFields = append(specServerServiceFields, "labels")
var specConfigurationIstioEnabledFields = []string{specField, "configuration", "istio", "enabled"}
var specConfigurationRuntimeEncryptionSecret = []string{specField, "configuration", "model", "runtimeEncryptionSecret"}
var specDomainSourceType = []string{specField, "domainHomeSourceType"}
var specConfigurationWDTConfigMap = []string{specField, "configuration", "model", "configMap"}
var specConfigurationDomainOnPVConfigMap = []string{specField, "configuration", "initializeDomainOnPV", "domain", "domainCreationConfigMap"}
var specConfigurationInitializeDomainOnPV = []string{specField, "configuration", "initializeDomainOnPV"}
var specMonitoringExporterFields = []string{specField, "monitoringExporter"}
var specRestartVersionFields = []string{specField, "restartVersion"}
var specServerStartPolicyFields = []string{specField, "serverStartPolicy"}
var specLogHomeFields = []string{specField, "logHome"}
var specLogHomeEnabledFields = []string{specField, "logHomeEnabled"}
var specLogHomeLayoutFields = []string{specField, "logHomeLayout"}

// this struct allows us to extract information from the unstructured WebLogic spec,
// so we can interface with the FLUENTD code
type containersMountsVolumes struct {
	Containers   []corev1.Container
	Volumes      []corev1.Volume
	VolumeMounts []corev1.VolumeMount
}

// Reconciler reconciles a VerrazzanoWebLogicWorkload object
type Reconciler struct {
	client.Client
	Log     *zap.SugaredLogger
	Scheme  *runtime.Scheme
	Metrics *metricstrait.Reconciler
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vzapi.VerrazzanoWebLogicWorkload{}).
		Complete(r)
}

// Reconcile reconciles a VerrazzanoWebLogicWorkload resource. It fetches the embedded WebLogic Domain CR, mutates it to add
// scopes and traits, and then writes out the CR (or deletes it if the workload is being deleted).
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanoweblogicworkloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanoweblogicworkloads/status,verbs=get;update;patch
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if ctx == nil {
		return ctrl.Result{}, errors.New("context cannot be nil")
	}

	// We do not want any resource to get reconciled if it is in namespace kube-system
	// This is due to a bug found in OKE, it should not affect functionality of any vz operators
	// If this is the case then return success
	if req.Namespace == vzconst.KubeSystem {
		log := zap.S().With(vzlogInit.FieldResourceNamespace, req.Namespace, vzlogInit.FieldResourceName, req.Name, vzlogInit.FieldController, controllerName)
		log.Infof("Weblogic workload resource %v should not be reconciled in kube-system namespace, ignoring", req.NamespacedName)
		return reconcile.Result{}, nil
	}

	// fetch the workload and unwrap the WebLogic resource
	workload, err := r.fetchWorkload(ctx, req.NamespacedName, zap.S())
	if err != nil {
		return clusters.IgnoreNotFoundWithLog(err, zap.S())
	}
	log, err := clusters.GetResourceLogger("verrazzanoweblogicworkload", req.NamespacedName, workload)
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for weblogic workload resource: %v", err)
		return clusters.NewRequeueWithDelay(), nil
	}
	log.Oncef("Reconciling WebLogic workload resource %v, generation %v", req.NamespacedName, workload.Generation)

	res, err := r.doReconcile(ctx, workload, log)
	if clusters.ShouldRequeue(res) {
		return res, nil
	}
	// Never return an error since it has already been logged. We don't want the
	// controller runtime to log again (with stack trace).  Just re-queue if there is an error.
	if err != nil {
		return clusters.NewRequeueWithDelay(), nil
	}

	log.Oncef("Finished reconciling WebLogic workload %v", req.NamespacedName)

	return ctrl.Result{}, nil
}

// doReconcile performs the reconciliation operations for the WebLogic workload
func (r *Reconciler) doReconcile(ctx context.Context, workload *vzapi.VerrazzanoWebLogicWorkload, log vzlog.VerrazzanoLogger) (ctrl.Result, error) {
	// Make sure the last generation exists in the status
	result, err := r.ensureLastGeneration(workload)
	if err != nil || result.Requeue {
		return result, err
	}

	u, err := r.initializeDomain(workload, log)
	if err != nil {
		return reconcile.Result{}, err
	}

	cus, err := r.initializeClusters(workload, log)
	if err != nil {
		return reconcile.Result{}, err
	}

	var existingDomain unstructured.Unstructured
	existingDomain.SetAPIVersion(u.GetAPIVersion())
	existingDomain.SetKind(u.GetKind())
	domainExists := true
	domainKey := types.NamespacedName{Name: u.GetName(), Namespace: workload.Namespace}
	if err := r.Get(ctx, domainKey, &existingDomain); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Debug("No existing domain found")
			domainExists = false
		} else {
			log.Errorf("Failed trying to obtain an existing domain: %v", err)
			return reconcile.Result{}, err
		}
	}

	// If the domain already exists, make sure that the domain can be restarted.
	// If the domain cannot be restarted, don't make any domain changes.
	if domainExists && !r.isOkToRestartWebLogic(workload) {
		log.Debug("There have been no changes to the WebLogic workload, nor has the restart annotation changed. The Domain will not be modified.")
		return ctrl.Result{}, nil
	}

	// Add the Fluentd sidecar container required for logging to the Domain.  If the image is old, update it
	if err = r.addLogging(ctx, log, workload, u); err != nil {
		return reconcile.Result{}, err
	}

	// Add logging traits to the Domain if they exist
	if err = r.addLoggingTrait(ctx, log, workload, u); err != nil {
		return reconcile.Result{}, err
	}

	// Add the monitoringExporter to the spec if not already present
	if err = addDefaultMonitoringExporter(u); err != nil {
		return reconcile.Result{}, err
	}

	// The istio.enabled field is no longer needed for v9 domains
	if isV8(u) {
		// Get the namespace resource that the VerrazzanoWebLogicWorkload resource is deployed to
		namespace := &corev1.Namespace{}
		if err = r.Client.Get(ctx, client.ObjectKey{Namespace: "", Name: workload.Namespace}, namespace); err != nil {
			return reconcile.Result{}, err
		}

		// Set the domain resource configuration.istio.enabled value
		if err = updateIstioEnabled(namespace.Labels, u); err != nil {
			return reconcile.Result{}, err
		}
	}

	// create the RuntimeEncryptionSecret if specified and the secret does not exist
	secret, found, err := unstructured.NestedString(u.Object, specConfigurationRuntimeEncryptionSecret...)
	if err != nil {
		return reconcile.Result{}, err
	}
	if found {
		nspace, _, err := unstructured.NestedString(u.Object, metadataField, "namespace")
		if err != nil {
			return reconcile.Result{}, err
		}
		err = r.createRuntimeEncryptionSecret(ctx, log, nspace, secret, workload.ObjectMeta.Labels)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Set/Update the WDT config map with WeblogicPluginEnabled setting
	if err = r.CreateOrUpdateWDTConfigMap(ctx, log, workload.Namespace, u, workload.ObjectMeta.Labels); err != nil {
		return reconcile.Result{}, err
	}

	// Create or update Cluster resources
	for i := range cus {
		if err = r.createOrUpdateResource(ctx, workload, log, cus[i], func(specCopy interface{}) error {
			if err := unstructured.SetNestedField(cus[i].Object, specCopy, specField); err != nil {
				return err
			}

			return nil
		}); err != nil {
			log.Errorf("Failed creating or updating WebLogic cluster CR: %v", err)
			return reconcile.Result{}, err
		}
	}

	// Create or update Domain resource
	if err = r.createOrUpdateResource(ctx, workload, log, u, func(specCopy interface{}) error {
		// Set the new Domain spec fields from the copy first, so we can overlay the lifecycle fields/annotations after,
		// otherwise they will be lost
		if err := unstructured.SetNestedField(u.Object, specCopy, specField); err != nil {
			return err
		}
		// If the domain already exists set any fields related to restart
		if domainExists {
			err = setDomainLifecycleFields(log, workload, u)
			if err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		log.Errorf("Failed creating or updating WebLogic domain CR: %v", err)
		return reconcile.Result{}, err
	}

	if err = r.updateStatusReconcileDone(ctx, workload); err != nil {
		return reconcile.Result{}, err
	}

	log.Debug("Successfully reconcile the WebLogic workload")
	return reconcile.Result{}, nil
}

func (r *Reconciler) initializeDomain(workload *vzapi.VerrazzanoWebLogicWorkload, log vzlog.VerrazzanoLogger) (*unstructured.Unstructured, error) {
	var u unstructured.Unstructured
	err := r.initializeResource(&u, workload, &workload.Spec.Template, log)
	if err != nil {
		return nil, err
	}

	if u.GetAPIVersion() == "" {
		u.SetAPIVersion(APIVersionV8)
	}
	u.SetKind(DomainKind)

	return &u, nil
}

func (r *Reconciler) initializeClusters(workload *vzapi.VerrazzanoWebLogicWorkload, log vzlog.VerrazzanoLogger) ([]*unstructured.Unstructured, error) {
	var clus []*unstructured.Unstructured
	for i := range workload.Spec.Clusters {
		var u unstructured.Unstructured
		err := r.initializeResource(&u, workload, &workload.Spec.Clusters[i], log)
		if err != nil {
			return nil, err
		}

		if u.GetAPIVersion() == "" {
			u.SetAPIVersion(APIVersionV1)
		}
		u.SetKind(ClusterKind)
		clus = append(clus, &u)
	}

	return clus, nil
}

func (r *Reconciler) initializeResource(u *unstructured.Unstructured, workload *vzapi.VerrazzanoWebLogicWorkload, resource *vzapi.VerrazzanoWebLogicWorkloadTemplate, log vzlog.VerrazzanoLogger) error {
	spec, err := vznav.ConvertRawExtensionToUnstructured(&resource.Spec)
	if err != nil {
		return err
	}

	if resource.APIVersion != "" {
		u.SetAPIVersion(resource.APIVersion)
	}

	if u.Object == nil {
		u.Object = make(map[string]interface{})
	}
	if err = unstructured.SetNestedField(u.Object, spec.Object, specField); err != nil {
		return err
	}

	metadata, err := vznav.ConvertRawExtensionToUnstructured(&resource.Metadata)
	if err != nil {
		return err
	}

	if err = unstructured.SetNestedField(u.Object, metadata.Object, metadataField); err != nil {
		return err
	}

	// make sure the namespace is set to the namespace of the component
	if err = unstructured.SetNestedField(u.Object, workload.Namespace, metadataField, "namespace"); err != nil {
		return err
	}

	// mutate the WebLogic domain resource, copy labels, add logging, etc.
	if err = copyLabels(log, workload.ObjectMeta.GetLabels(), u); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) createOrUpdateResource(ctx context.Context, workload *vzapi.VerrazzanoWebLogicWorkload, log vzlog.VerrazzanoLogger, u *unstructured.Unstructured, f func(interface{}) error) error {
	// make a copy of the WebLogic spec since u.Object will get overwritten in CreateOrUpdate
	// if the WebLogic CR exists
	specCopy, _, err := unstructured.NestedFieldCopy(u.Object, specField)
	if err != nil {
		log.Errorf("Failed to make a copy of the WebLogic spec: %v", err)
		return err
	}

	// set controller reference so the WebLogic domain CR gets deleted when the workload is deleted
	if err = controllerutil.SetControllerReference(workload, u, r.Scheme); err != nil {
		log.Errorf("Failed to set controller ref: %v", err)
		return err
	}

	if y, err := yaml.Marshal(u); err != nil {
		log.Debugf("Resource in raw format: %s ", u)
	} else {
		log.Debugf("Resource in YAML format: %s", string(y))
	}

	// write out the WebLogic resource
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, u, func() error {
		return f(specCopy)
	})
	if err != nil {
		log.Errorf("Failed creating or updating WebLogic CR: %v", err)
		return err
	}

	return nil
}

// fetchWorkload fetches the VerrazzanoWebLogicWorkload data given a namespaced name
func (r *Reconciler) fetchWorkload(ctx context.Context, name types.NamespacedName, log *zap.SugaredLogger) (*vzapi.VerrazzanoWebLogicWorkload, error) {
	var workload vzapi.VerrazzanoWebLogicWorkload
	if err := r.Get(ctx, name, &workload); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Debugf("VerrazzanoWebLogicWorkload %s has been deleted", name.Name)
		} else {
			log.Errorf("Failed to fetch VerrazzanoWebLogicWorkload %s: %v", name.Name, err)
		}
		return nil, err
	}

	return &workload, nil
}

// Make sure that the last generation exists in the status
func (r *Reconciler) ensureLastGeneration(wl *vzapi.VerrazzanoWebLogicWorkload) (ctrl.Result, error) {
	if len(wl.Status.LastGeneration) > 0 {
		return ctrl.Result{}, nil
	}

	// Update the status generation and always requeue
	wl.Status.LastGeneration = strconv.Itoa(int(wl.Generation))
	err := r.Status().Update(context.TODO(), wl)
	return ctrl.Result{Requeue: true, RequeueAfter: 1}, err
}

// Make sure that it is OK to restart WebLogic
func (r *Reconciler) isOkToRestartWebLogic(wl *vzapi.VerrazzanoWebLogicWorkload) bool {
	// Check if user created or changed the restart of lifecycle annotation
	if wl.Annotations != nil {
		if wl.Annotations[vzconst.RestartVersionAnnotation] != wl.Status.LastRestartVersion {
			return true
		}
		if wl.Annotations[vzconst.LifecycleActionAnnotation] != wl.Status.LastLifecycleAction {
			return true
		}
	}
	if wl.Status.LastGeneration != strconv.Itoa(int(wl.Generation)) {
		// The spec has changed ok to restart
		return true
	}
	// nothing in the spec or lifecycle annotations has changed
	return false
}

func isV8(weblogic *unstructured.Unstructured) bool {
	return weblogic.GetAPIVersion() == APIVersionV8
}

// copyLabels copies specific labels from the Verrazzano workload to the contained WebLogic resource
func copyLabels(log vzlog.VerrazzanoLogger, workloadLabels map[string]string, weblogic *unstructured.Unstructured) error {
	// the WebLogic domain spec/serverPod/labels field has labels that get propagated to the pods
	labels, found, _ := unstructured.NestedStringMap(weblogic.Object, specServerPodLabelsFields...)
	if !found {
		labels = map[string]string{}
	}

	// the WebLogic domain spec/serverService/labels field has labels that get propagated to the service
	svcLabels, found, _ := unstructured.NestedStringMap(weblogic.Object, specServerServiceLabelsFields...)
	if !found {
		svcLabels = map[string]string{}
	}

	// copy the oam component and app name labels
	if componentName, ok := workloadLabels[oam.LabelAppComponent]; ok {
		labels[oam.LabelAppComponent] = componentName
		svcLabels[oam.LabelAppComponent] = componentName
	}

	if appName, ok := workloadLabels[oam.LabelAppName]; ok {
		labels[oam.LabelAppName] = appName
		svcLabels[oam.LabelAppName] = appName
	}

	// Set the label indicating this is WebLogic workload
	labels[constants.LabelWorkloadType] = constants.WorkloadTypeWeblogic

	err := unstructured.SetNestedStringMap(weblogic.Object, labels, specServerPodLabelsFields...)
	if err != nil {
		log.Errorf("Failed to set labels in spec serverPod: %v", err)
		return err
	}
	err = unstructured.SetNestedStringMap(weblogic.Object, labels, specServerServiceLabelsFields...)
	if err != nil {
		log.Errorf("Failed to set labels in spec serverService: %v", err)
		return err
	}
	return nil
}

// addLogging adds a FLUENTD sidecar and updates the WebLogic spec if there is an associated LogInfo
// If the Fluentd image changed during an upgrade, then the new image will be used
func (r *Reconciler) addLogging(ctx context.Context, log vzlog.VerrazzanoLogger, workload *vzapi.VerrazzanoWebLogicWorkload, weblogic *unstructured.Unstructured) error {
	// extract just enough of the WebLogic data into concrete types, so we can merge with
	// the FLUENTD data
	var extracted containersMountsVolumes
	if serverPod, found, _ := unstructured.NestedMap(weblogic.Object, specServerPodFields...); found {
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(serverPod, &extracted); err != nil {
			return errors.New("unable to extract containers, volumes, and volume mounts from WebLogic spec")
		}
	}

	name, found, _ := unstructured.NestedString(weblogic.Object, "metadata", "name")
	if !found {
		return errors.New("expected to find metadata name in WebLogic spec")
	}

	// get the existing logHome setting - if it's set we use it otherwise we'll generate a logs location
	// using an emptydir volume
	volumeMountPath := scratchVolMountPath
	volumeName := storageVolumeName
	foundVolumeMount := false
	logHome, _, _ := unstructured.NestedString(weblogic.Object, specLogHomeFields...)
	if logHome != "" {
		// find the existing volume mount for the logHome - the Fluentd volume mount needs to match
		for _, mount := range extracted.VolumeMounts {
			if strings.HasPrefix(logHome, mount.MountPath) {
				volumeMountPath = mount.MountPath
				volumeName = mount.Name
				foundVolumeMount = true
				break
			}
		}

		if !foundVolumeMount {
			// user specified logHome, but it's not on any volume, Fluentd sidecar won't be able to collect logs
			log.Info("Unable to find a volume mount for domain logHome, log collection will not work")
		}
	}
	_, logHomeEnabledSet, _ := unstructured.NestedBool(weblogic.Object, specLogHomeEnabledFields...)

	// fluentdPod starts with what's in the spec. We add in the FLUENTD things when Apply is
	// called on the fluentdManager
	fluentdPod := &logging.FluentdPod{
		Containers:   extracted.Containers,
		Volumes:      extracted.Volumes,
		VolumeMounts: extracted.VolumeMounts,
		LogPath:      getWLSLogPath(logHome, name),
		HandlerEnv:   getWlsSpecificContainerEnv(logHome, name),
	}
	fluentdManager := &logging.Fluentd{Context: ctx,
		Log:                    zap.S(),
		Client:                 r.Client,
		ParseRules:             WlsFluentdParsingRules,
		StorageVolumeName:      volumeName,
		StorageVolumeMountPath: volumeMountPath,
		WorkloadType:           workloadType,
	}

	// fluentdManager.Apply wants a QRR, but it only cares about the namespace (at least for
	// this use case)
	resource := vzapi.QualifiedResourceRelation{Namespace: workload.Namespace}

	// note that this call has the side effect of creating a FLUENTD config map if one
	// does not already exist in the namespace
	if err := fluentdManager.Apply(logging.NewLogInfo(), resource, fluentdPod); err != nil {
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
		log.Errorf("Failed to set serverPod containers: %v", err)
		return err
	}
	err = unstructured.SetNestedSlice(weblogic.Object, fluentdPodUnstructured["volumes"].([]interface{}), specServerPodVolumesFields...)
	if err != nil {
		log.Errorf("Failed to set serverPod volumes: %v", err)
		return err
	}
	err = unstructured.SetNestedField(weblogic.Object, fluentdPodUnstructured["volumeMounts"].([]interface{}), specServerPodVolumeMountsFields...)
	if err != nil {
		log.Errorf("Failed to set serverPod volumeMounts: %v", err)
		return err
	}

	// set logHome if it was not already specified in the domain spec
	if logHome == "" {
		err = unstructured.SetNestedField(weblogic.Object, getWLSLogHome(name), specLogHomeFields...)
		if err != nil {
			log.Errorf("Failed to set logHome: %v", err)
			return err
		}
	}
	// set logHomeEnabled if it was not already specified in the domain spec
	if !logHomeEnabledSet {
		err = unstructured.SetNestedField(weblogic.Object, true, specLogHomeEnabledFields...)
		if err != nil {
			log.Errorf("Failed to set logHomeEnabled: %v", err)
			return err
		}
	}

	if !isV8(weblogic) {
		err = unstructured.SetNestedField(weblogic.Object, "Flat", specLogHomeLayoutFields...)
		if err != nil {
			log.Errorf("Failed to set logHomeLayout: %v", err)
			return err
		}
	}

	return nil
}

// createRuntimeEncryptionSecret creates the runtimeEncryptionSecret specified in the domain spec if it does not exist.
func (r *Reconciler) createRuntimeEncryptionSecret(ctx context.Context, log vzlog.VerrazzanoLogger, namespaceName string, secretName string, workloadLabels map[string]string) error {
	appName, ok := workloadLabels[oam.LabelAppName]
	if !ok {
		return errors.New("OAM app name label missing from metadata, unable to create owner reference to appconfig")
	}

	// Create the secret if it does not already exist
	secret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespaceName, Name: secretName}, secret)
	if err != nil && k8serrors.IsNotFound(err) {
		thePassword, err := genPassword(128)
		if err != nil {
			return err
		}
		secret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespaceName,
				Name:      secretName,
			},
			Data: map[string][]byte{
				"password": []byte(thePassword),
			},
		}

		// Set the owner reference.
		appConfig := &v1alpha2.ApplicationConfiguration{}
		err = r.Get(context.TODO(), types.NamespacedName{Namespace: namespaceName, Name: appName}, appConfig)
		if err != nil {
			return err
		}
		err = controllerutil.SetControllerReference(appConfig, secret, r.Scheme)
		if err != nil {
			return err
		}

		log.Debugf("Creating secret %s:%s", namespaceName, secretName)
		err = r.Create(ctx, secret)
		if err != nil {
			return err
		}

	} else if err != nil {
		return err
	}
	log.Debugf("Secret %s:%s already exist", namespaceName, secretName)

	return nil
}

// Update the status field with life cyele information as needed
func (r *Reconciler) updateStatusReconcileDone(ctx context.Context, wl *vzapi.VerrazzanoWebLogicWorkload) error {
	update := false
	if wl.Status.LastGeneration != strconv.Itoa(int(wl.Generation)) {
		wl.Status.LastGeneration = strconv.Itoa(int(wl.Generation))
		update = true
	}
	if wl.Annotations != nil {
		if wl.Annotations[vzconst.RestartVersionAnnotation] != wl.Status.LastRestartVersion {
			wl.Status.LastRestartVersion = wl.Annotations[vzconst.RestartVersionAnnotation]
			update = true
		}
		if wl.Annotations[vzconst.LifecycleActionAnnotation] != wl.Status.LastLifecycleAction {
			wl.Status.LastLifecycleAction = wl.Annotations[vzconst.LifecycleActionAnnotation]
			update = true
		}
	}
	if update {
		return r.Status().Update(ctx, wl)
	}
	return nil
}

// GetConfigMapLocation gets the location of the configmap in the model.
func (r *Reconciler) GetConfigMapLocation(u *unstructured.Unstructured) ([]string, error) {
	//Find domainHomeSourceType
	domainHomeSourceType, exists, err := unstructured.NestedString(u.Object, specDomainSourceType...)
	if err != nil {
		return nil, err
	}
	if exists {
		if domainHomeSourceType == "PersistentVolume" {
			_, fnd, err := unstructured.NestedMap(u.Object, specConfigurationInitializeDomainOnPV...)
			if err != nil {
				return nil, err

			}
			if fnd {
				return specConfigurationDomainOnPVConfigMap, err
			}
		} else {
			return specConfigurationWDTConfigMap, err
		}
	}
	return specConfigurationWDTConfigMap, err
}

// CreateOrUpdateWDTConfigMap creates a default WDT config map with WeblogicPluginEnabled setting if the
// WDT config map is not specified in the WebLogic spec. Otherwise, it updates the specified WDT config map
// with WeblogicPluginEnabled setting if not already done.
func (r *Reconciler) CreateOrUpdateWDTConfigMap(ctx context.Context, log vzlog.VerrazzanoLogger, namespaceName string, u *unstructured.Unstructured, workloadLabels map[string]string) error {
	// Get the specified WDT config map name in the WebLogic spec
	var location, err = r.GetConfigMapLocation(u)
	if err != nil {
		log.Errorf("Failed to get location of WDT configMap from WebLogic spec: %v", err)
		return err
	}

	configMapName, found, err := unstructured.NestedString(u.Object, location...)
	if err != nil {
		log.Errorf("Failed to extract WDT configMap from WebLogic spec: %v", err)
		return err
	}
	if !found {
		domainUID, domainUIDFound, err := unstructured.NestedString(u.Object, specDomainUID...)
		if err != nil {
			log.Errorf("Failed to extract domainUID from the WebLogic spec: %v", err)
			return err
		}
		if !domainUIDFound {
			log.Errorf("Failed to find domainUID in WebLogic spec: %v", err)
			return errors.New("unable to find domainUID in WebLogic spec")
		}
		// Create a default WDT config map
		err = r.createDefaultWDTConfigMap(ctx, log, namespaceName, domainUID, workloadLabels)
		if err != nil {
			return err
		}
		// Set WDT config map field in WebLogic spec
		err = unstructured.SetNestedField(u.Object, getWDTConfigMapName(domainUID), location...)
		if err != nil {
			log.Errorf("Failed to set WDT config map in WebLogic spec: %v", err)
			return err
		}
	} else {
		configMap, err := r.getConfigMap(ctx, u.GetNamespace(), configMapName)
		if err != nil {
			return err
		}
		if configMap == nil {
			log.Errorf("Failed to find the specified WDT config map: %v", err)
			return err
		}
		// Update WDT configMap configuration to add default WLS plugin configuration
		v := configMap.Data[webLogicPluginConfigYamlKey]
		if v == "" {
			byt, err := yaml.JSONToYAML([]byte(defaultWDTConfigMapData))
			if err != nil {
				return err
			}
			if configMap.Data == nil {
				configMap.Data = map[string]string{}
			}
			configMap.Data[webLogicPluginConfigYamlKey] = string(byt)
			err = r.Client.Update(ctx, configMap)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// createDefaultWDTConfigMap creates a default WDT config map with WeblogicPluginEnabled setting.
func (r *Reconciler) createDefaultWDTConfigMap(ctx context.Context, log vzlog.VerrazzanoLogger, namespaceName string, domainName string, workloadLabels map[string]string) error {
	configMapName := getWDTConfigMapName(domainName)
	// Create a configMap resource that will contain WeblogicPluginEnabled setting
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespaceName,
			Labels: map[string]string{
				webLogicDomainUIDLabel: domainName,
			},
		},
	}
	// Create the config map if it does not already exist
	configMapFound := &corev1.ConfigMap{}
	log.Debugf("Checking if WDT ConfigMap %s:%s exists", namespaceName, configMapName)
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespaceName}, configMapFound)
	if err != nil && k8serrors.IsNotFound(err) {
		// set controller reference so the WDT config map gets deleted when the app config is deleted
		appName, ok := workloadLabels[oam.LabelAppName]
		if !ok {
			return errors.New("OAM app name label missing from metadata, unable to create WDT config map")
		}
		appConfig := &v1alpha2.ApplicationConfiguration{}
		err = r.Get(context.TODO(), types.NamespacedName{Namespace: namespaceName, Name: appName}, appConfig)
		if err != nil {
			return err
		}
		if err = controllerutil.SetControllerReference(appConfig, configMap, r.Scheme); err != nil {
			log.Errorf("Failed to set controller ref for WDT config map: %v", err)
			return err
		}
		byt, err := yaml.JSONToYAML([]byte(defaultWDTConfigMapData))
		if err != nil {
			return err
		}
		configMap.Data = map[string]string{webLogicPluginConfigYamlKey: string(byt)}
		log.Debugf("Creating WDT ConfigMap %s:%s", namespaceName, configMapName)
		err = r.Create(ctx, configMap)
		if err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	}
	log.Debugf("ConfigMap %s:%s already exists", namespaceName, configMapName)
	return nil
}

// getConfigMap will get the ConfigMap for the given name
func (r *Reconciler) getConfigMap(ctx context.Context, namespace string, configMapName string) (*corev1.ConfigMap, error) {
	var wdtConfigMap = &corev1.ConfigMap{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespace}, wdtConfigMap)
	if err != nil {
		return nil, err
	}
	return wdtConfigMap, nil
}

// updateIstioEnabled sets the domain resource configuration.istio.enabled value based
// on the namespace label istio-injection
func updateIstioEnabled(labels map[string]string, u *unstructured.Unstructured) error {
	istioEnabled := false
	value, ok := labels["istio-injection"]
	if ok && value == "enabled" {
		istioEnabled = true
	}

	return unstructured.SetNestedField(u.Object, istioEnabled, specConfigurationIstioEnabledFields...)
}

func genPassword(passSize int) (string, error) {
	const passwordChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, passSize)
	for i := 0; i < passSize; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordChars))))
		if err != nil {
			return "", err
		}
		result[i] = passwordChars[num.Int64()]
	}
	return string(result), nil
}

// addDefaultMonitoringExporter adds monitoringExporter to the WebLogic spec if there is not one present
func addDefaultMonitoringExporter(weblogic *unstructured.Unstructured) error {
	if _, found, _ := unstructured.NestedFieldNoCopy(weblogic.Object, specMonitoringExporterFields...); !found {
		defaultMonitoringExporter, err := getDefaultMonitoringExporter()
		if err != nil {
			return err
		}
		err = unstructured.SetNestedField(weblogic.Object, defaultMonitoringExporter, specMonitoringExporterFields...)
		if err != nil {
			return err
		}
	}
	return nil
}

func getDefaultMonitoringExporter() (interface{}, error) {
	// get ImageSetting
	imageSetting := ""
	if value := os.Getenv("WEBLOGIC_MONITORING_EXPORTER_IMAGE"); len(value) > 0 {
		imageSetting = fmt.Sprintf("\"image\": \"%s\",\n    ", value)
	}

	// Create the buffer and the cluster issuer data struct
	templateData := defaultMonitoringExporterTemplateData{
		ImageSetting: imageSetting,
	}

	// Parse the template string and create the template object
	templ, err := template.New("defaultMonitoringExporter").Parse(defaultMonitoringExporterTemplate)
	if err != nil {
		return nil, err
	}

	// Execute the template object with the given data
	var buff bytes.Buffer
	err = templ.Execute(&buff, &templateData)
	if err != nil {
		return nil, err
	}

	var monitoringExporter map[string]interface{}
	err = json.Unmarshal(buff.Bytes(), &monitoringExporter)
	if err != nil {
		return nil, err
	}
	result, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&monitoringExporter)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// addLoggingTrait adds the logging trait sidecar to the workload
func (r *Reconciler) addLoggingTrait(ctx context.Context, log vzlog.VerrazzanoLogger, workload *vzapi.VerrazzanoWebLogicWorkload, weblogic *unstructured.Unstructured) error {
	loggingTrait, err := vznav.LoggingTraitFromWorkloadLabels(ctx, r.Client, log, workload.GetNamespace(), workload.ObjectMeta)
	if err != nil {
		return err
	}
	if loggingTrait == nil {
		return nil
	}
	configMapName := loggingNamePart + "-" + weblogic.GetName() + "-" + strings.ToLower(weblogic.GetKind())
	configMap := &corev1.ConfigMap{}
	err = r.Get(ctx, client.ObjectKey{Namespace: weblogic.GetNamespace(), Name: loggingNamePart + "-" + weblogic.GetName() + "-" + strings.ToLower(weblogic.GetKind())}, configMap)
	if err != nil && k8serrors.IsNotFound(err) {
		data := make(map[string]string)
		data["custom.conf"] = loggingTrait.Spec.LoggingConfig
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: weblogic.GetNamespace(),
				Labels:    weblogic.GetLabels(),
			},
			Data: data,
		}
		err = controllerutil.SetControllerReference(workload, configMap, r.Scheme)
		if err != nil {
			return err
		}
		log.Debugf("Creating logging trait configmap %s:%s", weblogic.GetNamespace(), loggingNamePart+"-"+weblogic.GetName()+"-"+strings.ToLower(weblogic.GetKind()))
		err = r.Create(ctx, configMap)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	log.Debugf("logging trait configmap %s:%s already exist", weblogic.GetNamespace(), loggingNamePart+"-"+weblogic.GetName()+"-"+strings.ToLower(weblogic.GetKind()))

	// extract just enough of the WebLogic data into concrete types, so we can merge with
	// the logging trait data
	var extract containersMountsVolumes
	if serverPod, found, _ := unstructured.NestedMap(weblogic.Object, specServerPodFields...); found {
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(serverPod, &extract); err != nil {
			return errors.New("unable to extract containers, volumes, and volume mounts from WebLogic spec")
		}
	}
	extracted := &containersMountsVolumes{
		Containers:   extract.Containers,
		VolumeMounts: extract.VolumeMounts,
		Volumes:      extract.Volumes,
	}
	loggingVolumeMount := &corev1.VolumeMount{
		MountPath: loggingMountPath,
		Name:      configMapName,
		SubPath:   loggingKey,
		ReadOnly:  true,
	}
	vmIndex := -1
	for i, vm := range extracted.VolumeMounts {
		if reflect.DeepEqual(vm, *loggingVolumeMount) {
			vmIndex = i
		}
	}
	if vmIndex != -1 {
		extracted.VolumeMounts[vmIndex] = *loggingVolumeMount
	} else {
		extracted.VolumeMounts = append(extracted.VolumeMounts, *loggingVolumeMount)
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
		VolumeMounts:    extracted.VolumeMounts,
		Env:             []corev1.EnvVar{*envFluentd},
	}
	cIndex := -1
	for i, c := range extracted.Containers {
		if c.Name == loggingNamePart {
			cIndex = i
		}
	}
	if cIndex != -1 {
		extracted.Containers[cIndex] = *loggingContainer
	} else {
		extracted.Containers = append(extracted.Containers, *loggingContainer)
	}

	loggingVolume := &corev1.Volume{
		Name: configMapName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
				DefaultMode: func(mode int32) *int32 {
					return &mode
				}(defaultMode),
			},
		},
	}
	vIndex := -1
	for i, v := range extracted.Volumes {
		if v.Name == configMapName {
			vIndex = i
		}
	}
	if vIndex != -1 {
		extracted.Volumes[vIndex] = *loggingVolume
	} else {
		extracted.Volumes = append(extracted.Volumes, *loggingVolume)
	}

	// convert the containers, volumes, and mounts in extracted to unstructured and set
	// the values in the spec
	extractedUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&extracted)
	if err != nil {
		return err
	}

	err = unstructured.SetNestedSlice(weblogic.Object, extractedUnstructured["containers"].([]interface{}), specServerPodContainersFields...)
	if err != nil {
		log.Errorf("Failed to set serverPod containers: %v", err)
		return err
	}
	err = unstructured.SetNestedSlice(weblogic.Object, extractedUnstructured["volumes"].([]interface{}), specServerPodVolumesFields...)
	if err != nil {
		log.Errorf("Failed to set serverPod volumes: %v", err)
		return err
	}

	return nil
}

// If any domainlifecycle start, stop, or restart is requested, then set the appropriate field in the domain resource
// Note that it is valid to a have new restartVersion value along with a lifecycle action change.  This
// will not result in additional restarts.
func setDomainLifecycleFields(log vzlog.VerrazzanoLogger, wl *vzapi.VerrazzanoWebLogicWorkload, domain *unstructured.Unstructured) error {
	if len(wl.Annotations[vzconst.LifecycleActionAnnotation]) > 0 && wl.Annotations[vzconst.LifecycleActionAnnotation] != wl.Status.LastLifecycleAction {
		action := wl.Annotations[vzconst.LifecycleActionAnnotation]
		if strings.EqualFold(action, vzconst.LifecycleActionStart) {
			return startWebLogicDomain(log, domain)
		}
		if strings.EqualFold(action, vzconst.LifecycleActionStop) {
			return stopWebLogicDomain(log, domain)
		}
	}
	if wl.Annotations != nil && wl.Annotations[vzconst.RestartVersionAnnotation] != wl.Status.LastRestartVersion {
		return restartWebLogic(log, domain, wl.Annotations[vzconst.RestartVersionAnnotation])
	}
	return nil
}

// Set domain restart version.  If it is changed from the previous value, then the WebLogic Operator will restart the domain
func restartWebLogic(log vzlog.VerrazzanoLogger, domain *unstructured.Unstructured, version string) error {
	err := unstructured.SetNestedField(domain.Object, version, specRestartVersionFields...)
	if err != nil {
		log.Errorf("Failed setting restartVersion in domain: %v", err)
		return err
	}
	return nil
}

// Set the serverStartPolicy to stop WebLogic domain, return the current serverStartPolicy
func stopWebLogicDomain(log vzlog.VerrazzanoLogger, domain *unstructured.Unstructured) error {
	never := Never
	ifneeded := IfNeeded
	if isV8(domain) {
		never = NeverV8
		ifneeded = IfNeededV8
	}

	// Return if serverStartPolicy is already never
	currentServerStartPolicy, _, _ := unstructured.NestedString(domain.Object, specServerStartPolicyFields...)
	if currentServerStartPolicy == never {
		return nil
	}

	// Save the last policy so that it can be used when starting the domain
	if len(currentServerStartPolicy) == 0 {
		currentServerStartPolicy = ifneeded
	}
	annos, found, err := unstructured.NestedStringMap(domain.Object, metaAnnotationFields...)
	if err != nil {
		log.Errorf("Failed getting domain annotations: %v", err)
		return err
	}
	if !found {
		annos = map[string]string{}
	}
	annos[lastServerStartPolicyAnnotation] = currentServerStartPolicy
	err = unstructured.SetNestedStringMap(domain.Object, annos, metaAnnotationFields...)
	if err != nil {
		log.Errorf("Failed to set annotations in domain: %v", err)
		return err
	}

	// set serverStartPolicy to "NEVER" to shut down the domain
	err = unstructured.SetNestedField(domain.Object, never, specServerStartPolicyFields...)
	if err != nil {
		log.Errorf("Failed to set serverStartPolicy in domain: %v", err)
		return err
	}
	return nil
}

// Set the serverStartPolicy to start the WebLogic domain
func startWebLogicDomain(log vzlog.VerrazzanoLogger, domain *unstructured.Unstructured) error {
	var startPolicy = IfNeeded
	if isV8(domain) {
		startPolicy = IfNeededV8
	}

	// Get the last serverStartPolicy if it exists
	annos, found, err := unstructured.NestedStringMap(domain.Object, metaAnnotationFields...)
	if err != nil {
		log.Errorf("Failed getting domain annotations: %v", err)
		return err
	}
	if found {
		oldPolicy := annos[lastServerStartPolicyAnnotation]
		if len(oldPolicy) > 0 {
			startPolicy = oldPolicy
		}
	}
	err = unstructured.SetNestedField(domain.Object, startPolicy, specServerStartPolicyFields...)
	if err != nil {
		return err
	}
	return nil
}

// getWDTConfigMapName builds a WDT config map name given a domain name
func getWDTConfigMapName(domainName string) string {
	return fmt.Sprintf("%s%s", domainName, WDTConfigMapNameSuffix)
}
