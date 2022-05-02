package overrides

import (
	"context"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateConfigMap(configMap *v1.ConfigMap) error {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		ginkgov2.Fail(err.Error())
	}

	kubeconfig, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		ginkgov2.Fail(err.Error())
	}

	clientset, err := k8sutil.GetKubernetesClientsetWithConfig(kubeconfig)
	if err != nil {
		ginkgov2.Fail(err.Error())
	}

	configMap, err = clientset.CoreV1().ConfigMaps(configMap.Namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func UpdateConfigMap(namespace string, name string, data []byte) {
	configMap, err := pkg.GetConfigMap(name, namespace)
	if err != nil {
		ginkgov2.Fail(err.Error())
	}

	configMap.Data["values1.yaml"] = string(data)

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		ginkgov2.Fail(err.Error())
	}

	kubeconfig, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		ginkgov2.Fail(err.Error())
	}

	clientset, err := k8sutil.GetKubernetesClientsetWithConfig(kubeconfig)
	if err != nil {
		ginkgov2.Fail(err.Error())
	}

	clientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})

}
