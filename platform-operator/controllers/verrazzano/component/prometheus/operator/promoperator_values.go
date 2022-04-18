package operator

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	installNamespace = "verrazzano-install"
	resourceName     = "po-values"
)

// This is the function that I'm thinking will be called by the helm component doing the install.
// We can process the value data in whatever fashion is the best. As a placeholder
// I have set this func to return a string
func getValues(cr *vzapi.Verrazzano) (string, error) {
	if cr.Spec.Components.PrometheusOperator.Values != nil {
		// finish code and return
	}

	if cr.Spec.Components.PrometheusOperator.ValuesRefs != nil {
		if cr.Spec.Components.PrometheusOperator.ValuesRefs.ConfigMap != nil {
			configMap, err := getConfigMap(installNamespace, resourceName)
			if err != nil {
				return "", err
			}
		}
		// Go through key list and set value acc to precedence
		if cr.Spec.Components.PrometheusOperator.ValuesRefs.Secret != nil {
			secret, err := getSecret(installNamespace, resourceName)
			if err != nil {
				return "", err
			}
			// Go through key list and set value acc to precedence
		}
	}
	return "", nil
}

func getConfigMap(namespace string, configMapName string) (*v1.ConfigMap, error) {
	// Get the ConfigMap and Secret from the clientset, then fetch values.yaml
	// Maybe the controller can include a lister to list resources
	clientset, err := k8sutil.GetKubernetesClientset()

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return configMap, nil
}

func getSecret(namespace string, secretName string) (*v1.Secret, error) {
	clientset, err := k8sutil.GetKubernetesClientset()

	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return secret, nil
}
