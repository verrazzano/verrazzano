package overrides

import (
	"context"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func CreateConfigMap(configMap *v1.ConfigMap) {
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

	clientset.CoreV1().ConfigMaps(configMap.Namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
}

func GetConfigMap(namespace string, name string) (*v1.ConfigMap, error) {
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

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginkgov2.Fail(err.Error())
	}

	return configMap, nil
}

func UpdateConfigMap(namespace string, name string, data []byte) {
	configMap, err := GetConfigMap(namespace, name)
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

func GetDeployment(nsn types.NamespacedName) (*appsv1.Deployment, error) {
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

	deployment, err := clientset.AppsV1().Deployments(nsn.Namespace).Get(context.TODO(), nsn.Name, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return deployment, err
}
