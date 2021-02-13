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
	clientset, err := getMCClient(&secret)
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to get the client for cluster %q", clusterName))
		return
	}

	// Start the thread for syncing multi-cluster objects
	go StartSync(clientset, log)
}

// StartSync - start the thread for syncing multi-cluster objects
func StartSync(clientset client.Client, log logr.Logger) {
	log.Info("Starting sync of multi-cluster objects")
	ctx := context.TODO()

	// Periodically loop looking for multi-cluster objects
	for {
		err := syncMCSecretObjects(clientset, ctx, log)
		if err != nil {
			log.Error(err, "Error syncing MultiClusterSecret objects")
		}
		err = syncMCConfigMapObjects(clientset, ctx, log)
		if err != nil {
			log.Error(err, "Error syncing MultiClusterConfigMap objects")
		}
		err = syncMCLoggingScopeObjects(clientset, ctx, log)
		if err != nil {
			log.Error(err, "Error syncing MultiClusterLoggingScope objects")
		}
		err = syncMCComponentObjects(clientset, ctx, log)
		if err != nil {
			log.Error(err, "Error syncing MultiClusterComponent objects")
		}
		err = syncMCApplicationConfigurationObjects(clientset, ctx, log)
		if err != nil {
			log.Error(err, "Error syncing MultiClusterApplicationConfiguration objects")
		}
		time.Sleep(1 * time.Minute)
	}
}

// Synchronize MultiClusterSecret objects to the local cluster
func syncMCSecretObjects(clientset client.Client, ctx context.Context, log logr.Logger) error {
	// Get all the MultiClusterSecret objects from the admin cluster
	allMCSecrets := &clustersv1alpha1.MultiClusterSecretList{}
	err := clientset.List(ctx, allMCSecrets)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the records that are targetted to this cluster
	for _, mcSecert := range allMCSecrets.Items {
		_, err := createOrUpdateMCSecret(clientset, ctx, mcSecert)
		if err != nil {
			return err
		}
	}
	return nil
}

// Create or update a MultiClusterSecret
func createOrUpdateMCSecret(clientset client.Client, ctx context.Context, mcSecret clustersv1alpha1.MultiClusterSecret) (controllerutil.OperationResult, error) {
	var mcSecretNew clustersv1alpha1.MultiClusterSecret
	mcSecretNew.Namespace = mcSecret.Namespace
	mcSecretNew.Name = mcSecret.Name

	return controllerutil.CreateOrUpdate(ctx, clientset, &mcSecretNew, func() error {
		mutateMCSecret(mcSecret, &mcSecretNew)
		return nil
	})
}

// mutateMCSecret mutates the corev1.Secret to reflect the contents of the parent MultiClusterSecret
func mutateMCSecret(mcSecret clustersv1alpha1.MultiClusterSecret, mcSecretNew *clustersv1alpha1.MultiClusterSecret) {
	mcSecretNew.Spec.Placement = mcSecret.Spec.Placement
	mcSecretNew.Spec.Template = mcSecret.Spec.Template
}

// Synchronize MultiClusterConfigMap objects to the local cluster
func syncMCConfigMapObjects(clientset client.Client, ctx context.Context, log logr.Logger) error {
	// Get all the MultiClusterConfigMap objects from the admin cluster
	allMCConfigMaps := &clustersv1alpha1.MultiClusterConfigMapList{}
	err := clientset.List(ctx, allMCConfigMaps)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

// Synchronize MultiClusterComponent objects to the local cluster
func syncMCComponentObjects(clientset client.Client, ctx context.Context, log logr.Logger) error {
	// Get all the MultiClusterComponent objects from the admin cluster
	allMCComponents := &clustersv1alpha1.MultiClusterComponentList{}
	err := clientset.List(ctx, allMCComponents)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

// Synchronize MultiClusterLoggingScope objects to the local cluster
func syncMCLoggingScopeObjects(clientset client.Client, ctx context.Context, log logr.Logger) error {
	// Get all the MultiClusterLoggingScope objects from the admin cluster
	allMCLoggingScopes := &clustersv1alpha1.MultiClusterLoggingScopeList{}
	err := clientset.List(ctx, allMCLoggingScopes)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

// Synchronize MultiClusterApplicationConfiguration objects to the local cluster
func syncMCApplicationConfigurationObjects(clientset client.Client, ctx context.Context, log logr.Logger) error {
	// Get all the MultiClusterApplicationConfiguration objects from the admin cluster
	allMCApplicationConfigurations := &clustersv1alpha1.MultiClusterApplicationConfigurationList{}
	err := clientset.List(ctx, allMCApplicationConfigurations)
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
		return fmt.Errorf("The secret named %s in namespace %s is missing the required field cluster-name", secret.Name, secret.Namespace)
	}

	// The secret must contain a kubeconfig
	_, ok = secret.Data["kubeconfig"]
	if !ok {
		return fmt.Errorf("The secret named %s in namespace %s is missing the required field kubeconfig", secret.Name, secret.Namespace)
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
