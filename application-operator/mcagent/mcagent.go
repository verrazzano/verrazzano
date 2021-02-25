// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/go-logr/logr"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StartAgent - start the agent thread for syncing multi-cluster objects
func StartAgent(client client.Client, log logr.Logger) {
	// Wait for the existence of the verrazzano-cluster secret.  It contains the credentials
	// for connecting to a managed cluster.
	log.Info("Starting multi-cluster agent")
	secret := corev1.Secret{}

	for {
		err := client.Get(context.TODO(), types.NamespacedName{Name: constants.MCRegistrationSecret, Namespace: constants.VerrazzanoSystemNamespace}, &secret)
		if err != nil {
			time.Sleep(60 * time.Second)
		} else {
			err := validateClusterSecret(&secret)
			if err != nil {
				log.Error(err, "Secret validation failed")
			} else {
				break
			}
		}
	}

	// The cluster secret exists
	managedClusterName := string(secret.Data[constants.ClusterNameData])
	log.Info(fmt.Sprintf("Found secret named %s in namespace %s for cluster named %q", secret.Name, secret.Namespace, managedClusterName))

	// Create the client for accessing the admin cluster
	adminClient, err := getAdminClient(&secret)
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to get the client for cluster %q", managedClusterName))
		return
	}

	// Create the synchronization context structure
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        client,
		Log:                log,
		ManagedClusterName: managedClusterName,
		Context:            context.TODO(),
		ProjectNamespaces:  []string{},
	}

	// Start syncing multi-cluster objects
	s.StartSync()
}

// StartSync - start syncing multi-cluster objects
func (s *Syncer) StartSync() {
	s.Log.Info("Starting sync of multi-cluster objects")

	// Periodically loop looking for multi-cluster objects
	for {
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

		time.Sleep(1 * time.Minute)
	}
}

// Validate the cluster secret
func validateClusterSecret(secret *corev1.Secret) error {
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
