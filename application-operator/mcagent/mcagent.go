// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ENV VAR for registration secret version
const registrationSecretVersion = "REGISTRATION_SECRET_VERSION"

// StartAgent - start the agent thread for syncing multi-cluster objects
func StartAgent(client client.Client, statusUpdateChannel chan clusters.StatusUpdateMessage, log logr.Logger) {
	// Wait for the existence of the verrazzano-cluster-agent secret.  It contains the credentials
	// for connecting to a managed cluster.
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
			s.Log.Error(err, "error processing multi-cluster resources")
		}
		s.configureBeats()
		if !s.AgentReadyToSync() {
			// there is no admin cluster we are connected to, so nowhere to send any status updates
			// received - discard them
			discardStatusMessages(s.StatusUpdateChannel)
		}
		time.Sleep(60 * time.Second)
	}
}

// ProcessAgentThread - process one iteration of the agent thread
func (s *Syncer) ProcessAgentThread() error {
	secret := corev1.Secret{}

	// Get the secret
	err := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: constants.MCAgentSecret, Namespace: constants.VerrazzanoSystemNamespace}, &secret)
	if err != nil {
		if clusters.IgnoreNotFoundWithLog("secret", err, s.Log) == nil && s.AgentSecretFound {
			s.Log.Info(fmt.Sprintf("the secret %s in namespace %s was deleted", constants.MCAgentSecret, constants.VerrazzanoSystemNamespace))
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
		s.Log.Info(fmt.Sprintf("Found secret named %s in namespace %s, cluster name changed from %q to %q", secret.Name, secret.Namespace, s.ManagedClusterName, managedClusterName))
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

	err = s.UpdateVerrazzanoManagedClusterStatus(managedClusterName)
	if err != nil {
		return fmt.Errorf("Failed to update vmc status for vmc %swith error %v", managedClusterName, err)
	}
	// Update the status of our VMC on the admin cluster to record the last time we connected
	err = s.updateVMCStatus()
	if err != nil {
		// we couldn't update status of the VMC - but we should keep going with the rest of the work
		s.Log.Error(err, "Failed to update VMC status on admin cluster")
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
	vmc.Status.LastAgentConnectTime = metav1.Now()

	// update status of VMC
	return s.AdminClient.Status().Update(s.Context, &vmc)
}

// SyncMultiClusterResources - sync multi-cluster objects
func (s *Syncer) SyncMultiClusterResources() {
	err := s.syncVerrazzanoProjects()
	if err != nil {
		s.Log.Error(err, "Error syncing VerrazzanoProject objects")
	}

	// Synchronize objects one namespace at a time
	for _, namespace := range s.ProjectNamespaces {
		err = s.syncMCSecretObjects(namespace)
		if err != nil {
			s.Log.Error(err, "Error syncing MultiClusterSecret objects")
		}
		err = s.syncMCConfigMapObjects(namespace)
		if err != nil {
			s.Log.Error(err, "Error syncing MultiClusterConfigMap objects")
		}
		err = s.syncMCLoggingScopeObjects(namespace)
		if err != nil {
			s.Log.Error(err, "Error syncing MultiClusterLoggingScope objects")
		}
		err = s.syncMCComponentObjects(namespace)
		if err != nil {
			s.Log.Error(err, "Error syncing MultiClusterComponent objects")
		}
		err = s.syncMCApplicationConfigurationObjects(namespace)
		if err != nil {
			s.Log.Error(err, "Error syncing MultiClusterApplicationConfiguration objects")
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

	clientset, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

// reconfigure beats by restarting Verrazzano Operator deployment if ManagedClusterName has been changed
func (s *Syncer) configureBeats() {
	// Get the verrazzano-operator deployment
	deploymentName := types.NamespacedName{Name: "verrazzano-operator", Namespace: constants.VerrazzanoSystemNamespace}
	deployment := appsv1.Deployment{}
	err := s.LocalClient.Get(context.TODO(), deploymentName, &deployment)
	if err != nil {
		s.Log.Info(fmt.Sprintf("Failed to find the deployment %s, %s", deploymentName, err.Error()))
		return
	}
	if len(deployment.Spec.Template.Spec.Containers) < 1 {
		s.Log.Info(fmt.Sprintf("No container defined in the deployment %s", deploymentName))
		return
	}

	// get the cluster name
	secretVersion := ""
	regSecret := corev1.Secret{}
	regErr := s.LocalClient.Get(context.TODO(), types.NamespacedName{Name: constants.MCRegistrationSecret, Namespace: constants.VerrazzanoSystemNamespace}, &regSecret)
	if regErr != nil {
		if clusters.IgnoreNotFoundWithLog("secret", regErr, s.Log) != nil {
			return
		}
	} else {
		secretVersion = regSecret.ResourceVersion
	}
	secretVersionEnv := getEnvValue(deployment.Spec.Template.Spec.Containers[0].Env, registrationSecretVersion)

	// CreateOrUpdate updates the deployment if cluster name or es secret version changed
	if secretVersionEnv != secretVersion {
		controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &deployment, func() error {
			s.Log.Info(fmt.Sprintf("Update the deployment %s, registration secret version from %q to %q", deploymentName, secretVersionEnv, secretVersion))
			// update the container env
			env := updateEnvValue(deployment.Spec.Template.Spec.Containers[0].Env, registrationSecretVersion, secretVersion)
			deployment.Spec.Template.Spec.Containers[0].Env = env
			return nil
		})
	}
}

func getEnvValue(envs []corev1.EnvVar, envName string) string {
	for _, env := range envs {
		if env.Name == envName {
			return env.Value
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
