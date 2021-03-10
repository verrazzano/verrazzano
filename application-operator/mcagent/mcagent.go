// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

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

const managedClusterNameEnvName = "MANAGED_CLUSTER_NAME"

// StartAgent - start the agent thread for syncing multi-cluster objects
func StartAgent(client client.Client, log logr.Logger) {
	// Wait for the existence of the verrazzano-cluster-agent secret.  It contains the credentials
	// for connecting to a managed cluster.
	log.Info("Starting multi-cluster agent")

	// Initialize the syncer object
	s := &Syncer{
		LocalClient:           client,
		Log:                   log,
		Context:               context.TODO(),
		ProjectNamespaces:     []string{},
		ManagedClusterName:    "",
		AgentSecretFound:      false,
		SecretResourceVersion: "",
	}

	for {
		// Process one iteration of the agent thread
		err := s.ProcessAgentThread()
		if err != nil {
			s.Log.Error(err, "error processing multi-cluster resources")
		}
		s.configureBeats()
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
			s.ManagedClusterName = ""
		}
		return nil
	}
	err = validateAgentSecret(&secret)
	if err != nil {
		return fmt.Errorf("secret validation failed: %v", err)
	}

	// Remember the secret had been found in order to notice if it gets deleted
	s.AgentSecretFound = true

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

	// Sync multi-cluster objects
	s.SyncMultiClusterResources()
	return nil
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
	if err == nil {
		if len(deployment.Spec.Template.Spec.Containers) > 0 {
			clusterNameEnv := getClusterNameEnvValue(deployment.Spec.Template.Spec.Containers[0].Env)
			if clusterNameEnv != s.ManagedClusterName {
				s.Log.Info(fmt.Sprintf("Restarting the deployment %s, cluster name changed from %q to %q", deploymentName, clusterNameEnv, s.ManagedClusterName))
				controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &deployment, func() error {
					deployment.Spec.Template.Spec.Containers[0].Env = updateClusterNameEnvValue(deployment.Spec.Template.Spec.Containers[0].Env, s.ManagedClusterName)
					return nil
				})
			}
		} else {
			s.Log.Info(fmt.Sprintf("No container defined in the deployment %s", deploymentName))
		}
	} else {
		s.Log.Info(fmt.Sprintf("Failed to find the deployment %s, %s", deploymentName, err.Error()))
	}
}

// get the value for env var MANAGED_CLUSTER_NAME
func getClusterNameEnvValue(envs []corev1.EnvVar) string {
	for _, env := range envs {
		if env.Name == managedClusterNameEnvName {
			return env.Value
		}
	}
	return ""
}

// update the value for env var MANAGED_CLUSTER_NAME
func updateClusterNameEnvValue(envs []corev1.EnvVar, newValue string) []corev1.EnvVar {
	for i, env := range envs {
		if env.Name == managedClusterNameEnvName {
			envs[i].Value = newValue
			return envs
		}
	}
	return append(envs, corev1.EnvVar{Name: managedClusterNameEnvName, Value: newValue})
}
