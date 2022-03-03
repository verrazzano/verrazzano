// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ENV VAR for registration secret version
const registrationSecretVersion = "REGISTRATION_SECRET_VERSION"

// StartAgent - start the agent thread for syncing multi-cluster objects
func StartAgent(client client.Client, statusUpdateChannel chan clusters.StatusUpdateMessage, log *zap.SugaredLogger) {
	// Wait for the existence of the verrazzano-cluster-agent secret.  It contains the credentials
	// for connecting to a managed cluster.
	log = log.With(vzlog.FieldAgent, "multi-cluster")
	log.Info("Starting multi-cluster agent")

	// Initialize the syncer object
	s := &Syncer{
		LocalClient:           client,
		Log:                   log,
		Context:               context.TODO(),
		ProjectNamespaces:     []string{},
		AgentSecretFound:      false,
		SecretResourceVersion: "",
		StatusUpdateChannel:   statusUpdateChannel,
	}

	for {
		// Process one iteration of the agent thread
		err := s.ProcessAgentThread()
		if err != nil {
			s.Log.Errorf("Failed processing multi-cluster resources: %v", err)
		}
		s.updateDeployment("verrazzano-monitoring-operator")
		s.configureLogging()
		if !s.AgentReadyToSync() {
			// there is no admin cluster we are connected to, so nowhere to send any status updates
			// received - discard them
			discardStatusMessages(s.StatusUpdateChannel)
		}
		time.Sleep(vzconstants.VMCAgentPollingTimeInterval)
	}
}

// ProcessAgentThread - process one iteration of the agent thread
func (s *Syncer) ProcessAgentThread() error {
	secret := corev1.Secret{}

	// Get the secret
	err := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: constants.MCAgentSecret, Namespace: constants.VerrazzanoSystemNamespace}, &secret)
	if err != nil {
		if client.IgnoreNotFound(err) == nil && s.AgentSecretFound {
			s.Log.Debugf("the secret %s in namespace %s was deleted", constants.MCAgentSecret, constants.VerrazzanoSystemNamespace)
			s.AgentSecretFound = false
			s.AgentSecretValid = false
		}
		return nil
	}
	err = validateAgentSecret(&secret)
	if err != nil {
		s.AgentSecretValid = false
		return fmt.Errorf("secret validation failed: %v", err)
	}

	// Remember the secret had been found in order to notice if it gets deleted
	s.AgentSecretFound = true
	s.AgentSecretValid = true

	// The cluster secret exists - log the cluster name only if it changes
	managedClusterName := string(secret.Data[constants.ClusterNameData])
	if managedClusterName != s.ManagedClusterName {
		s.Log.Debugf("Found secret named %s in namespace %s, cluster name changed from %q to %q", secret.Name, secret.Namespace, s.ManagedClusterName, managedClusterName)
		s.ManagedClusterName = managedClusterName

	}

	// Create the client for accessing the admin cluster when there is a change in the secret
	if secret.ResourceVersion != s.SecretResourceVersion {
		adminClient, err := getAdminClient(&secret)
		if err != nil {
			return fmt.Errorf("Failed to get the client for cluster %q with error %v", managedClusterName, err)
		}
		s.AdminClient = adminClient
		s.SecretResourceVersion = secret.ResourceVersion
	}

	// Update the status of our VMC on the admin cluster to record the last time we connected
	err = s.updateVMCStatus()
	if err != nil {
		// we couldn't update status of the VMC - but we should keep going with the rest of the work
		s.Log.Errorf("Failed to update VMC status on admin cluster: %v", err)
	}

	// Sync multi-cluster objects
	s.SyncMultiClusterResources()
	return nil
}

func (s *Syncer) updateVMCStatus() error {
	vmcName := client.ObjectKey{Name: s.ManagedClusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}
	vmc := platformopclusters.VerrazzanoManagedCluster{}
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
		return fmt.Errorf("Failed to get api prometheus host for vmc %s with error %v", vmcName, err)
	}

	vmc.Status.PrometheusHost = prometheusHost

	// update status of VMC
	return s.AdminClient.Status().Update(s.Context, &vmc)
}

// SyncMultiClusterResources - sync multi-cluster objects
func (s *Syncer) SyncMultiClusterResources() {
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
	_, ok = secret.Data[constants.AdminKubeconfigData]
	if !ok {
		return fmt.Errorf("the secret named %s in namespace %s is missing the required field %s", secret.Name, secret.Namespace, constants.AdminKubeconfigData)
	}

	return nil
}

// Get the clientset for accessing the admin cluster
func getAdminClient(secret *corev1.Secret) (client.Client, error) {
	// Create a temp file that contains the kubeconfig
	tmpFile, err := ioutil.TempFile("", "kubeconfig")
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(tmpFile.Name(), secret.Data[constants.AdminKubeconfigData], 0600)
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
	_ = platformopclusters.AddToScheme(scheme)
	_ = oamv1alpha2.SchemeBuilder.AddToScheme(scheme)
	_ = corev1.SchemeBuilder.AddToScheme(scheme)

	clientset, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

// reconfigure deployment if cluster registration has been changed
func (s *Syncer) updateDeployment(name string) {
	// Get the deployment
	deploymentName := types.NamespacedName{Name: name, Namespace: constants.VerrazzanoSystemNamespace}
	deployment := appsv1.Deployment{}
	err := s.LocalClient.Get(context.TODO(), deploymentName, &deployment)
	if err != nil {
		s.Log.Errorf("Failed to find the deployment %s: %v", deploymentName, err)
		return
	}
	if len(deployment.Spec.Template.Spec.Containers) < 1 {
		s.Log.Debugf("No container defined in the deployment %s", deploymentName)
		return
	}

	// get the cluster name
	secretVersion := ""
	regSecret := corev1.Secret{}
	regErr := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: constants.MCRegistrationSecret, Namespace: constants.VerrazzanoSystemNamespace}, &regSecret)
	if regErr != nil {
		if client.IgnoreNotFound(regErr) != nil {
			return
		}
	} else {
		secretVersion = regSecret.ResourceVersion
	}
	secretVersionEnv := getEnvValue(&deployment.Spec.Template.Spec.Containers, registrationSecretVersion)

	// CreateOrUpdate updates the deployment if cluster registration secret version changed
	if secretVersionEnv != secretVersion {
		controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &deployment, func() error {
			s.Log.Debugf("Update the deployment %s, registration secret version from %q to %q", deploymentName, secretVersionEnv, secretVersion)
			// update the container env
			env := updateEnvValue(deployment.Spec.Template.Spec.Containers[0].Env, registrationSecretVersion, secretVersion)
			deployment.Spec.Template.Spec.Containers[0].Env = env
			return nil
		})
	}
}

// reconfigure Fluentd by restarting Fluentd DaemonSet if ManagedClusterName has been changed
func (s *Syncer) configureLogging() {
	loggingName := types.NamespacedName{Name: vzconstants.FluentdDaemonSetName, Namespace: constants.VerrazzanoSystemNamespace}
	daemonSet := appsv1.DaemonSet{}
	err := s.LocalClient.Get(context.TODO(), loggingName, &daemonSet)
	if err != nil {
		s.Log.Errorf("Failed to find the logging DaemonSet %s: %v", loggingName, err)
		return
	}

	regSecret := corev1.Secret{}
	regErr := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: constants.MCRegistrationSecret, Namespace: constants.VerrazzanoSystemNamespace}, &regSecret)
	if regErr != nil {
		if client.IgnoreNotFound(regErr) != nil {
			return
		}
	}

	// CreateOrUpdate updates the fluentd daemonset - if no changes to the daemonset after we mutate it in memory,
	// controllerutil will not update it
	controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &daemonSet, func() error {
		s.updateLoggingDaemonSet(regSecret, &daemonSet)
		return nil
	})
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

func (s *Syncer) updateLoggingDaemonSet(regSecret corev1.Secret, ds *appsv1.DaemonSet) {
	isManaged := false
	if regSecret.ResourceVersion != "" && regSecret.Data != nil {
		isManaged = true
	}

	vzESURL, vzESSecret, err := s.getVzESURLSecret()
	if err != nil {
		return
	}

	ds.Spec.Template.Spec.Volumes = updateLoggingDaemonsetVolumes(isManaged, vzESSecret, ds.Spec.Template.Spec.Volumes)
	for i, c := range ds.Spec.Template.Spec.Containers {
		if c.Name == "fluentd" {
			ds.Spec.Template.Spec.Containers[i].Env = updateLoggingDaemonsetEnv(regSecret, isManaged, vzESURL, vzESSecret, ds.Spec.Template.Spec.Containers[i].Env)
		}
	}
}

const (
	defaultClusterName   = constants.DefaultClusterName
	defaultElasticURL    = "http://verrazzano-authproxy-elasticsearch:8775"
	defaultSecretName    = "verrazzano-es-internal" //nolint:gosec //#gosec G101
	esConfigMapName      = "fluentd-es-config"
	esConfigMapURLKey    = "es-url"
	esConfigMapSecretKey = "es-secret"
)

func updateLoggingDaemonsetEnv(regSecret corev1.Secret, isManaged bool, vzESURL, vzESSecret string, old []corev1.EnvVar) []corev1.EnvVar {
	var esSecretName string
	var esURL string
	var clusterName string
	var usernameKey string
	var passwordKey string
	if isManaged {
		esSecretName = constants.MCRegistrationSecret
		esURL = string(regSecret.Data[constants.ElasticsearchURLData])
		clusterName = string(regSecret.Data[constants.ClusterNameData])
		usernameKey = constants.ElasticsearchUsernameData
		passwordKey = constants.ElasticsearchPasswordData
	} else {
		esSecretName = vzESSecret
		esURL = vzESURL
		clusterName = defaultClusterName
		usernameKey = constants.VerrazzanoUsernameData
		passwordKey = constants.VerrazzanoPasswordData
	}

	var new []corev1.EnvVar
	for _, env := range old {
		if env.Name == constants.FluentdClusterNameEnvVar {
			new = append(new, corev1.EnvVar{
				Name:  env.Name,
				Value: clusterName,
			})
		} else if env.Name == constants.FluentdElasticsearchURLEnvVar {
			new = append(new, corev1.EnvVar{
				Name:  env.Name,
				Value: esURL,
			})
		} else if env.Name == constants.FluentdElasticsearchUserEnvVar {
			new = append(new, corev1.EnvVar{
				Name: env.Name,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: esSecretName,
						},
						Key: usernameKey,
						Optional: func(opt bool) *bool {
							return &opt
						}(true),
					},
				},
			})
		} else if env.Name == constants.FluentdElasticsearchPwdEnvVar {
			new = append(new, corev1.EnvVar{
				Name: env.Name,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: esSecretName,
						},
						Key: passwordKey,
						Optional: func(opt bool) *bool {
							return &opt
						}(true),
					},
				},
			})
		} else {
			new = append(new, env)
		}
	}
	return new
}

func updateLoggingDaemonsetVolumes(isManaged bool, vzESSecret string, old []corev1.Volume) []corev1.Volume {
	secretName := constants.MCRegistrationSecret
	if !isManaged {
		secretName = vzESSecret
	}
	var new []corev1.Volume
	for _, vol := range old {
		if vol.Name == "secret-volume" {
			new = append(new, corev1.Volume{
				Name: vol.Name,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: secretName},
				},
			})
		} else {
			new = append(new, vol)
		}
	}
	return new
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
		return "", fmt.Errorf("Unable to fetch ingress %s/%s, %v", constants.VerrazzanoSystemNamespace, constants.VzConsoleIngress, err)
	}
	return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host), nil
}

// GetPrometheusHost returns the prometheus host for Verrazzano instance.
func (s *Syncer) GetPrometheusHost() (string, error) {
	ingress := &networkingv1.Ingress{}
	err := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: constants.VzPrometheusIngress, Namespace: constants.VerrazzanoSystemNamespace}, ingress)
	if err != nil {
		return "", fmt.Errorf("unable to fetch ingress %s/%s, %v", constants.VerrazzanoSystemNamespace, constants.VzPrometheusIngress, err)
	}
	return ingress.Spec.Rules[0].Host, nil
}

// getVzESURLSecret returns the elasticsearchURL and elasticsearchSecret from Verrazzano CR
func (s *Syncer) getVzESURLSecret() (string, string, error) {
	url := defaultElasticURL
	secret := defaultSecretName
	esConfig := corev1.ConfigMap{}
	err := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: esConfigMapName, Namespace: constants.VerrazzanoSystemNamespace}, &esConfig)
	if err != nil {
		s.Log.Errorf("Failed to find the ConfigMap %s/%s: %v", constants.VerrazzanoSystemNamespace, esConfigMapName, err)
	}
	if len(esConfig.Data[esConfigMapURLKey]) > 0 {
		url = esConfig.Data[esConfigMapURLKey]
	}
	if len(esConfig.Data[esConfigMapSecretKey]) > 0 {
		secret = esConfig.Data[esConfigMapSecretKey]
	}
	return url, secret, nil
}
