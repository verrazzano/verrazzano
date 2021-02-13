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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Syncer contains context for synchronize operations
type Syncer struct {
	AdminClient client.Client
	MCClient    client.Client
	Log         logr.Logger
	ClusterName string
	Context     context.Context
}

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
	clusterName := string(secret.Data["cluster-name"])
	log.Info(fmt.Sprintf("Found secret named %s in namespace %s for cluster named %q", secret.Name, secret.Namespace, clusterName))

	// Create the client for accessing the managed cluster
	mcClient, err := getMCClient(&secret)
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to get the client for cluster %q", clusterName))
		return
	}

	// Start the thread for syncing multi-cluster objects
	s := &Syncer{
		AdminClient: mcClient,
		MCClient:    client,
		Log:         log,
		ClusterName: secret.ClusterName,
		Context:     context.TODO(),
	}

	go s.StartSync()
}

// StartSync - start the thread for syncing multi-cluster objects
func (s *Syncer) StartSync() {
	s.Log.Info("Starting sync of multi-cluster objects")

	// Periodically loop looking for multi-cluster objects
	for {
		err := s.syncMCSecretObjects()
		if err != nil {
			s.Log.Error(err, "Error syncing MultiClusterSecret objects")
		}
		err = s.syncMCConfigMapObjects()
		if err != nil {
			s.Log.Error(err, "Error syncing MultiClusterConfigMap objects")
		}
		err = s.syncMCLoggingScopeObjects()
		if err != nil {
			s.Log.Error(err, "Error syncing MultiClusterLoggingScope objects")
		}
		err = s.syncMCComponentObjects()
		if err != nil {
			s.Log.Error(err, "Error syncing MultiClusterComponent objects")
		}
		err = s.syncMCApplicationConfigurationObjects()
		if err != nil {
			s.Log.Error(err, "Error syncing MultiClusterApplicationConfiguration objects")
		}
		time.Sleep(1 * time.Minute)
	}
}

// Synchronize MultiClusterSecret objects to the local cluster
func (s *Syncer) syncMCSecretObjects() error {
	// Get all the MultiClusterSecret objects from the admin cluster
	allMCSecrets := &clustersv1alpha1.MultiClusterSecretList{}
	listOptions := &client.ListOptions{}
	err := s.AdminClient.List(s.Context, allMCSecrets, listOptions)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the records that are targeted to this cluster
	for _, mcSecert := range allMCSecrets.Items {
		_, err := s.createOrUpdateMCSecret(mcSecert)
		if err != nil {
			return err
		}
	}
	return nil
}

// Create or update a MultiClusterSecret
func (s *Syncer) createOrUpdateMCSecret(mcSecret clustersv1alpha1.MultiClusterSecret) (controllerutil.OperationResult, error) {
	var mcSecretNew clustersv1alpha1.MultiClusterSecret
	mcSecretNew.Namespace = mcSecret.Namespace
	mcSecretNew.Name = mcSecret.Name

	// Create or update on the local cluster
	return controllerutil.CreateOrUpdate(s.Context, s.MCClient, &mcSecretNew, func() error {
		mutateMCSecret(mcSecret, &mcSecretNew)
		return nil
	})
}

// mutateMCSecret mutates the MultiClusterSecret to reflect the contents of the parent MultiClusterSecret
func mutateMCSecret(mcSecret clustersv1alpha1.MultiClusterSecret, mcSecretNew *clustersv1alpha1.MultiClusterSecret) {
	mcSecretNew.Spec.Placement = mcSecret.Spec.Placement
	mcSecretNew.Spec.Template = mcSecret.Spec.Template
}

// Synchronize MultiClusterConfigMap objects to the local cluster
func (s *Syncer) syncMCConfigMapObjects() error {
	// Get all the MultiClusterConfigMap objects from the admin cluster
	allMCConfigMaps := &clustersv1alpha1.MultiClusterConfigMapList{}
	err := s.AdminClient.List(s.Context, allMCConfigMaps)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

// Synchronize MultiClusterComponent objects to the local cluster
func (s *Syncer) syncMCComponentObjects() error {
	// Get all the MultiClusterComponent objects from the admin cluster
	allMCComponents := &clustersv1alpha1.MultiClusterComponentList{}
	err := s.AdminClient.List(s.Context, allMCComponents)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the records that are targeted to this cluster
	for _, mcComponent := range allMCComponents.Items {
		_, err := s.createOrUpdateMCComponent(mcComponent)
		if err != nil {
			return err
		}
	}

	return nil
}

// Create or update a MultiClusterComponent
func (s *Syncer) createOrUpdateMCComponent(mcComponent clustersv1alpha1.MultiClusterComponent) (controllerutil.OperationResult, error) {
	var mcComponentNew clustersv1alpha1.MultiClusterComponent
	mcComponentNew.Namespace = mcComponent.Namespace
	mcComponentNew.Name = mcComponent.Name

	// Create or update on the local cluster
	return controllerutil.CreateOrUpdate(s.Context, s.MCClient, &mcComponentNew, func() error {
		mutateMCComponent(mcComponent, &mcComponentNew)
		return nil
	})
}

// mutateMCComponent mutates the MultiClusterComponent to reflect the contents of the parent MultiClusterComponent
func mutateMCComponent(mcComponent clustersv1alpha1.MultiClusterComponent, mcComponentNew *clustersv1alpha1.MultiClusterComponent) {
	mcComponentNew.Spec.Placement = mcComponent.Spec.Placement
	mcComponentNew.Spec.Template = mcComponent.Spec.Template
}

// Synchronize MultiClusterLoggingScope objects to the local cluster
func (s *Syncer) syncMCLoggingScopeObjects() error {
	// Get all the MultiClusterLoggingScope objects from the admin cluster
	allMCLoggingScopes := &clustersv1alpha1.MultiClusterLoggingScopeList{}
	err := s.AdminClient.List(s.Context, allMCLoggingScopes)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

// Synchronize MultiClusterApplicationConfiguration objects to the local cluster
func (s *Syncer) syncMCApplicationConfigurationObjects() error {
	// Get all the MultiClusterApplicationConfiguration objects from the admin cluster
	allMCApplicationConfigurations := &clustersv1alpha1.MultiClusterApplicationConfigurationList{}
	err := s.AdminClient.List(s.Context, allMCApplicationConfigurations)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

// Validate the cluster secret
func validateClusterSecret(secret *corev1.Secret) error {
	// The secret must contain a cluster-name
	_, ok := secret.Data["cluster-name"]
	if !ok {
		return fmt.Errorf("the secret named %s in namespace %s is missing the required field cluster-name", secret.Name, secret.Namespace)
	}

	// The secret must contain a kubeconfig
	_, ok = secret.Data["kubeconfig"]
	if !ok {
		return fmt.Errorf("the secret named %s in namespace %s is missing the required field kubeconfig", secret.Name, secret.Namespace)
	}

	return nil
}

// Get the clientset for accessing the managed cluster
func getMCClient(secret *corev1.Secret) (client.Client, error) {
	// Create a temp file that contains the kubeconfig
	tmpFile, err := ioutil.TempFile("/tmp", "kubeconfig")
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(tmpFile.Name(), secret.Data["kubeconfig"], 0777)
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
