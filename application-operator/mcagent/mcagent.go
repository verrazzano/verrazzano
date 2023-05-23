// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"
	"os"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ENV VAR for registration secret version
const (
	registrationSecretVersion = "REGISTRATION_SECRET_VERSION"
	cattleAgentHashData       = "cattle-agent-hash" // the data field name for the cattleAgentHash in the agent state configmap
	requeueDelayMinSeconds    = 50
	requeueDelayMaxSeconds    = 70
)

// Name of config map that stores mc agent state
var mcAgentStateConfigMapName = types.NamespacedName{Name: "mc-agent-state", Namespace: constants.VerrazzanoMultiClusterNamespace}

var getAdminClientFunc = createAdminClient

var mcAppConfCRDName = fmt.Sprintf("%s.%s", clustersv1alpha1.MultiClusterAppConfigResource, clustersv1alpha1.SchemeGroupVersion.Group)

// Reconciler reconciles one iteration of the Managed cluster agent
type Reconciler struct {
	client.Client
	Log          *zap.SugaredLogger
	Scheme       *runtime.Scheme
	AgentChannel chan clusters.StatusUpdateMessage
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		WithEventFilter(r.createAgentPredicate()).
		Complete(r)
}

func (r *Reconciler) createAgentPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return r.isAgentSecret(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return r.isAgentSecret(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return r.isAgentSecret(e.ObjectNew)
		},
	}
}

func (r *Reconciler) isAgentSecret(object client.Object) bool {
	return object.GetNamespace() == constants.VerrazzanoSystemNamespace && object.GetName() == constants.MCAgentSecret
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Get the agent secret
	agentSecret := corev1.Secret{}
	if err := r.Get(ctx, req.NamespacedName, &agentSecret); err != nil {
		// there is no admin cluster we are connected to, so nowhere to send any status updates
		// received - discard them
		discardStatusMessages(r.AgentChannel)
		return clusters.IgnoreNotFoundWithLog(err, r.Log)
	}
	if agentSecret.DeletionTimestamp != nil {
		r.Log.Debugf("the secret %v was deleted", req.NamespacedName)
		// there is no admin cluster we are connected to, so nowhere to send any status updates
		// received - discard them
		discardStatusMessages(r.AgentChannel)
		return clusters.NewRequeueWithRandomDelay(requeueDelayMinSeconds, requeueDelayMaxSeconds), nil
	}
	if err := validateAgentSecret(&agentSecret); err != nil {
		// agent secret is invalid - log and also discard status messages on the channel since there
		// is no valid admin cluster to send status updates to
		discardStatusMessages(r.AgentChannel)
		return clusters.NewRequeueWithRandomDelay(requeueDelayMinSeconds, requeueDelayMaxSeconds), fmt.Errorf("Agent secret validation failed: %v", err)
	}
	r.Log.Debug("Reconciling multi-cluster agent")

	// Process one iteration of the agent thread
	err := r.ProcessAgentThread(ctx, agentSecret)
	if err != nil {
		r.Log.Errorf("failed processing multi-cluster resources: %v", err)
	}
	return clusters.NewRequeueWithRandomDelay(requeueDelayMinSeconds, requeueDelayMaxSeconds), nil
}

// ProcessAgentThread - process one iteration of the agent thread
func (r *Reconciler) ProcessAgentThread(ctx context.Context, agentSecret corev1.Secret) error {
	managedClusterName := string(agentSecret.Data[constants.ClusterNameData])

	// Initialize the syncer object
	s := &Syncer{
		LocalClient:         r.Client,
		Log:                 r.Log,
		Context:             ctx,
		ProjectNamespaces:   []string{},
		StatusUpdateChannel: r.AgentChannel,
		ManagedClusterName:  managedClusterName,
	}

	// Read current agent state from config map
	mcAgentStateConfigMap := corev1.ConfigMap{Data: map[string]string{}}
	if err := r.Get(ctx, mcAgentStateConfigMapName, &mcAgentStateConfigMap); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get the agent state config map %v: %v", mcAgentStateConfigMapName, err)
	}

	// Create the client for accessing the admin cluster
	adminClient, err := getAdminClientFunc(&agentSecret)
	if err != nil {
		return fmt.Errorf("failed to get the client for cluster %q with error %v", managedClusterName, err)
	}
	s.AdminClient = adminClient

	// Sync cattle-cluster-agent deployment which will set the new cattleAgentHash on the Syncer
	cattleAgentHashValue, err := s.syncCattleClusterAgent(mcAgentStateConfigMap.Data[cattleAgentHashData], "")
	if err != nil {
		// we couldn't sync the cattle-cluster-agent - but we should keep going with the rest of the work
		r.Log.Errorf("Failed to synchronize cattle-cluster-agent: %v", err)
	}

	// Update mc-agent-state config map with the managed cluster name or cattle agent hash if needed
	if err := r.updateMCAgentStateConfigMap(ctx, managedClusterName, cattleAgentHashValue); err != nil {
		return err
	}

	// Update all Prometheus monitors relabel configs in all namespaces with new cluster name if needed
	err = s.updatePrometheusMonitorsClusterName()
	if err != nil {
		return fmt.Errorf("failed to update the cluster name to %s on Prometheus monitor resources with error %v", s.ManagedClusterName, err)
	}

	// Update the status of our VMC on the admin cluster to record the last time we connected
	err = s.updateVMCStatus()
	if err != nil {
		// we couldn't update status of the VMC - but we should keep going with the rest of the work
		r.Log.Errorf("Failed to update VMC status on admin cluster: %v", err)
	}

	// Sync multi-cluster objects
	s.SyncMultiClusterResources()

	// Delete the managed cluster resources if deregistration occurs
	err = s.syncDeregistration()
	if err != nil {
		// we couldn't delete the managed cluster resources - but we should keep going with the rest of the work
		r.Log.Errorf("Failed to sync the deregistration process: %v", err)
	}

	// Check whether the admin or local clusters' CA certs have rolled, and sync as necessary
	_, err = s.syncClusterCAs()
	if err != nil {
		// we couldn't sync the cluster CAs - but we should keep going with the rest of the work
		r.Log.Errorf("Failed to synchronize cluster CA certificates: %v", err)
	}

	return nil
}

// updateMCAgentStateConfigMap updates the managed cluster name and cattle agent hash in the
// agent state config map if those have changed from what was there before
func (r *Reconciler) updateMCAgentStateConfigMap(ctx context.Context, managedClusterName string, cattleAgentHashValue string) error {
	mcAgentStateConfigMap := corev1.ConfigMap{}
	mcAgentStateConfigMap.Name = mcAgentStateConfigMapName.Name
	mcAgentStateConfigMap.Namespace = mcAgentStateConfigMapName.Namespace
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, &mcAgentStateConfigMap, func() error {
		if mcAgentStateConfigMap.Data == nil {
			mcAgentStateConfigMap.Data = map[string]string{}
		}
		existingClusterName := mcAgentStateConfigMap.Data[constants.ClusterNameData]
		if existingClusterName != managedClusterName {
			// Log the cluster name only if it changes
			r.Log.Infof("Cluster name changed from '%q' to '%q', updating the agent state ConfigMap", existingClusterName, managedClusterName)
			mcAgentStateConfigMap.Data[constants.ClusterNameData] = managedClusterName
		}
		existingCattleAgentHash := mcAgentStateConfigMap.Data[cattleAgentHashData]
		if existingCattleAgentHash != cattleAgentHashValue {
			// Log that the cattle agent hash has changed
			r.Log.Infof("The %s has changed, updating the agent state ConfigMap", cattleAgentHashData)
			mcAgentStateConfigMap.Data[cattleAgentHashData] = cattleAgentHashValue
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update agent state in ConfigMap %v: %v", mcAgentStateConfigMapName, err)
	}
	return nil
}

func (s *Syncer) updateVMCStatus() error {
	vmcName := client.ObjectKey{Name: s.ManagedClusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}
	vmc := v1alpha1.VerrazzanoManagedCluster{}
	err := s.AdminClient.Get(s.Context, vmcName, &vmc)
	if err != nil {
		return err
	}

	curTime := metav1.Now()
	vmc.Status.LastAgentConnectTime = &curTime
	apiURL, err := s.GetAPIServerURL()
	if err != nil {
		return fmt.Errorf("Failed to get api server url for vmc %s with error %v", vmcName, err)
	}

	vmc.Status.APIUrl = apiURL
	prometheusHost, err := s.GetPrometheusHost()
	if err != nil {
		return fmt.Errorf("Failed to get api prometheus host to update VMC %s: %v", vmcName, err)
	}
	if prometheusHost != "" {
		vmc.Status.PrometheusHost = prometheusHost
	}

	// Get the Thanos API ingress URL from the local managed cluster, and populate
	// it in the VMC status on the admin cluster, so that admin cluster's Thanos query wire up
	// to the managed cluster
	thanosAPIHost, err := s.getThanosQueryStoreAPIHost()
	if err != nil {
		return fmt.Errorf("Failed to get Thanos query URL to update VMC %s: %v", vmcName, err)
	}

	// If Thanos is disabled, we want to empty the host so Prometheus federation returns
	vmc.Status.ThanosQueryStore = thanosAPIHost

	// update status of VMC
	return s.AdminClient.Status().Update(s.Context, &vmc)
}

// SyncMultiClusterResources - sync multi-cluster objects
func (s *Syncer) SyncMultiClusterResources() {
	// if the MultiClusterApplicationConfiguration CRD does not exist, the other MC resources are
	// unlikely to exist, and we don't need to sync the resources
	mcAppConfCRD := apiextv1.CustomResourceDefinition{}
	if err := s.LocalClient.Get(s.Context,
		types.NamespacedName{Name: mcAppConfCRDName}, &mcAppConfCRD); err != nil {
		if apierrors.IsNotFound(err) {
			s.Log.Debugf("CRD %s not found - skip syncing multicluster resources", mcAppConfCRDName)
			return
		}
		s.Log.Errorf("Failed retrieving CRD %s: %v", mcAppConfCRDName, err)
	}
	err := s.syncVerrazzanoProjects()
	if err != nil {
		s.Log.Errorf("Failed syncing VerrazzanoProject objects: %v", err)
	}

	// Synchronize objects one namespace at a time
	for _, namespace := range s.ProjectNamespaces {
		err = s.syncSecretObjects(namespace)
		if err != nil {
			s.Log.Errorf("Failed to sync Secret objects: %v", err)
		}
		err = s.syncMCSecretObjects(namespace)
		if err != nil {
			s.Log.Errorf("Failed to sync MultiClusterSecret objects: %v", err)
		}
		err = s.syncMCConfigMapObjects(namespace)
		if err != nil {
			s.Log.Errorf("Failed to sync MultiClusterConfigMap objects: %v", err)
		}
		err = s.syncMCComponentObjects(namespace)
		if err != nil {
			s.Log.Errorf("Failed to sync MultiClusterComponent objects: %v", err)
		}
		err = s.syncMCApplicationConfigurationObjects(namespace)
		if err != nil {
			s.Log.Errorf("Failed to sync MultiClusterApplicationConfiguration objects: %v", err)
		}

		s.processStatusUpdates()

	}
}

// Validate the agent secret
func validateAgentSecret(secret *corev1.Secret) error {
	// The secret must contain a cluster name
	_, ok := secret.Data[constants.ClusterNameData]
	if !ok {
		return fmt.Errorf("the secret named %s in namespace %s is missing the required field %s", secret.Name, secret.Namespace, constants.ClusterNameData)
	}

	// The secret must contain a kubeconfig
	_, ok = secret.Data[mcconstants.KubeconfigKey]
	if !ok {
		return fmt.Errorf("the secret named %s in namespace %s is missing the required field %s", secret.Name, secret.Namespace, mcconstants.KubeconfigKey)
	}

	return nil
}

// Get the clientset for accessing the admin cluster
func createAdminClient(secret *corev1.Secret) (client.Client, error) {
	// Create a temp file that contains the kubeconfig
	tmpFile, err := os.CreateTemp("", "kubeconfig")
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(tmpFile.Name(), secret.Data[mcconstants.KubeconfigKey], 0600)
	defer os.Remove(tmpFile.Name())
	if err != nil {
		return nil, err
	}

	config, err := clientcmd.BuildConfigFromFlags("", tmpFile.Name())
	if err != nil {
		return nil, err
	}
	scheme := runtime.NewScheme()
	_ = clustersv1alpha1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = oamv1alpha2.SchemeBuilder.AddToScheme(scheme)
	_ = corev1.SchemeBuilder.AddToScheme(scheme)

	clientset, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func getEnvValue(containers *[]corev1.Container, envName string) string {
	for _, container := range *containers {
		for _, env := range container.Env {
			if env.Name == envName {
				return env.Value
			}
		}
	}
	return ""
}

func updateEnvValue(envs []corev1.EnvVar, envName string, newValue string) []corev1.EnvVar {
	for i, env := range envs {
		if env.Name == envName {
			envs[i].Value = newValue
			return envs
		}
	}
	return append(envs, corev1.EnvVar{Name: envName, Value: newValue})
}

// discardStatusMessages discards all messages in the statusUpdateChannel - this will
// prevent the channel buffer from filling up in the case of a non-managed cluster
func discardStatusMessages(statusUpdateChannel chan clusters.StatusUpdateMessage) {
	length := len(statusUpdateChannel)
	for i := 0; i < length; i++ {
		<-statusUpdateChannel
	}
}

// GetAPIServerURL returns the API Server URL for Verrazzano instance.
func (s *Syncer) GetAPIServerURL() (string, error) {
	ingress := &networkingv1.Ingress{}
	err := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: constants.VzConsoleIngress, Namespace: constants.VerrazzanoSystemNamespace}, ingress)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("Unable to fetch ingress %s/%s, %v", constants.VerrazzanoSystemNamespace, constants.VzConsoleIngress, err)
	}
	return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host), nil
}

// GetPrometheusHost returns the prometheus host for Verrazzano instance.
func (s *Syncer) GetPrometheusHost() (string, error) {
	ingress := &networkingv1.Ingress{}
	err := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: constants.VzPrometheusIngress, Namespace: constants.VerrazzanoSystemNamespace}, ingress)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("unable to fetch ingress %s/%s, %v", constants.VerrazzanoSystemNamespace, constants.VzPrometheusIngress, err)
	}
	return ingress.Spec.Rules[0].Host, nil
}

// getThanosQueryStoreAPIHost returns the Thanos Query Store API Endpoint URL for Verrazzano instance.
func (s *Syncer) getThanosQueryStoreAPIHost() (string, error) {
	ingress := &networkingv1.Ingress{}
	err := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: vzconstants.ThanosQueryStoreIngress, Namespace: constants.VerrazzanoSystemNamespace}, ingress)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("unable to fetch ingress %s/%s, %v", constants.VerrazzanoSystemNamespace, constants.VzPrometheusIngress, err)
	}
	return ingress.Spec.Rules[0].Host, nil
}
