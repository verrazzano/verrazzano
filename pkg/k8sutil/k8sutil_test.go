// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package k8sutil_test

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"net/url"
	"os"
	"testing"

	spdyfake "github.com/verrazzano/verrazzano/pkg/k8sutil/fake"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	networkingv1 "k8s.io/api/networking/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/homedir"
)

const envVarHome = "HOME"
const dummyKubeConfig = "dummy-kubeconfig"
const dummyk8sHost = "http://localhost"
const appConfigName = "test"
const appConfigNamespace = "test"
const resultString = "{\"result\":\"result\"}"

func TestGetKubeConfigLocationEnvVarTestKubeconfig(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarTestKubeConfig)
	randomKubeConfig := "/home/testing/somerandompath"
	// Test using environment variable
	err := os.Setenv(k8sutil.EnvVarTestKubeConfig, randomKubeConfig)
	asserts.NoError(err)
	kubeConfigLoc, err := k8sutil.GetKubeConfigLocation()
	asserts.NoError(err)
	asserts.Equal(randomKubeConfig, kubeConfigLoc)
	// Reset env variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, prevEnvVar)
	asserts.NoError(err)

}

func TestGetKubeConfigLocationEnvVar(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarKubeConfig)
	randomKubeConfig := "/home/xyz/somerandompath"
	// Test using environment variable
	err := os.Setenv(k8sutil.EnvVarKubeConfig, randomKubeConfig)
	asserts.NoError(err)
	kubeConfigLoc, err := k8sutil.GetKubeConfigLocation()
	asserts.NoError(err)
	asserts.Equal(randomKubeConfig, kubeConfigLoc)
	// Reset env variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVar)
	asserts.NoError(err)

}
func TestGetKubeConfigLocationHomeDir(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Test without environment variable
	err := os.Setenv(k8sutil.EnvVarKubeConfig, "")
	asserts.NoError(err)
	kubeConfigLoc, err := k8sutil.GetKubeConfigLocation()
	asserts.NoError(err)
	asserts.Equal(kubeConfigLoc, homedir.HomeDir()+"/.kube/config")
	// Reset env variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVar)
	asserts.NoError(err)
}

func TestGetKubeConfigLocationReturnsError(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarHome := os.Getenv(envVarHome)
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset HOME environment variable
	err := os.Setenv(envVarHome, "")
	asserts.NoError(err)
	// Unset KUBECONFIG environment variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, "")
	asserts.NoError(err)
	_, err = k8sutil.GetKubeConfigLocation()
	asserts.Error(err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
	err = os.Setenv(envVarHome, prevEnvVarHome)
	asserts.NoError(err)
}

func TestGetKubeConfigReturnsError(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarHome := os.Getenv(envVarHome)
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset HOME environment variable
	err := os.Setenv(envVarHome, "")
	asserts.NoError(err)
	// Unset KUBECONFIG environment variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, "")
	asserts.NoError(err)
	_, err = k8sutil.GetKubeConfig()
	asserts.Error(err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
	err = os.Setenv(envVarHome, prevEnvVarHome)
	asserts.NoError(err)
}

func TestGetKubeConfigDummyKubeConfig(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)
	kubeconfig, err := k8sutil.GetKubeConfig()
	asserts.NoError(err)
	asserts.NotNil(kubeconfig)
	asserts.Equal(kubeconfig.Host, dummyk8sHost)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}

func TestGetKubernetesClientsetReturnsError(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarHome := os.Getenv(envVarHome)
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset HOME environment variable
	err := os.Setenv(envVarHome, "")
	asserts.NoError(err)
	// Unset KUBECONFIG environment variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, "")
	asserts.NoError(err)
	_, err = k8sutil.GetKubernetesClientset()
	asserts.Error(err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
	err = os.Setenv(envVarHome, prevEnvVarHome)
	asserts.NoError(err)
}

func TestGetKubernetesClientsetDummyKubeConfig(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)
	clientset, err := k8sutil.GetKubernetesClientset()
	asserts.NoError(err)
	asserts.NotNil(clientset)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}

func TestGetIstioClientsetReturnsError(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarHome := os.Getenv(envVarHome)
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset HOME environment variable
	err := os.Setenv(envVarHome, "")
	asserts.NoError(err)
	// Unset KUBECONFIG environment variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, "")
	asserts.NoError(err)
	_, err = k8sutil.GetIstioClientset()
	asserts.Error(err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
	err = os.Setenv(envVarHome, prevEnvVarHome)
	asserts.NoError(err)
}

func TestGetIstioClientsetDummyKubeConfig(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)
	istioClientSet, err := k8sutil.GetIstioClientset()
	asserts.NoError(err)
	asserts.NotNil(istioClientSet)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}

func TestGetHostnameFromGatewayReturnsErrorNoKubeconfig(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarHome := os.Getenv(envVarHome)
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset HOME environment variable
	err := os.Setenv(envVarHome, "")
	asserts.NoError(err)
	// Unset KUBECONFIG environment variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, "")
	asserts.NoError(err)
	_, err = k8sutil.GetHostnameFromGateway(appConfigNamespace, appConfigName)
	asserts.Error(err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
	err = os.Setenv(envVarHome, prevEnvVarHome)
	asserts.NoError(err)
}

func TestGetHostnameFromGatewayErrorListGateways(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)
	_, err = k8sutil.GetHostnameFromGateway(appConfigNamespace, appConfigName)
	asserts.Error(err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}

func TestGetHostnameFromGatewayGatewayForAppConfigDoesNotExist(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)

	gateway1 := istiov1alpha3.Gateway{}
	gateway1.Name = "test1"
	gateway2 := istiov1alpha3.Gateway{}
	gateway2.Name = "test2"
	hostname, _ := k8sutil.GetHostnameFromGateway(appConfigNamespace, appConfigName, gateway1, gateway2)
	asserts.Empty(hostname)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}

func TestGetHostnameFromGatewayGatewaysForAppConfigExists(t *testing.T) {
	asserts := assert.New(t)
	serverHost := "testhost"
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)

	gateway1 := istiov1alpha3.Gateway{}
	gateway1.Name = fmt.Sprintf("%s-%s-gw", appConfigNamespace, appConfigName)
	server := &v1alpha3.Server{}
	server.Hosts = append(server.Hosts, serverHost)
	gateway1.Spec.Servers = append(gateway1.Spec.Servers, server)

	gateway2 := istiov1alpha3.Gateway{}
	gateway2.Name = "test1"
	hostname, _ := k8sutil.GetHostnameFromGateway(appConfigNamespace, appConfigName, gateway1, gateway2)
	asserts.Equal(serverHost, hostname)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}

// TestExecPod tests running a command on a remote pod
// GIVEN a pod in a cluster and a command to run on that pod
//
//	WHEN ExecPod is called
//	THEN ExecPod return the stdout, stderr, and a nil error
func TestExecPod(t *testing.T) {
	k8sutil.NewPodExecutor = spdyfake.NewPodExecutor
	spdyfake.PodExecResult = func(url *url.URL) (string, string, error) { return resultString, "", nil }
	cfg, client := spdyfake.NewClientsetConfig()
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "name",
		},
	}
	stdout, _, err := k8sutil.ExecPod(client, cfg, pod, "container", []string{"run", "some", "command"})
	assert.Nil(t, err)
	assert.Equal(t, resultString, stdout)
}

// TestExecPodNoTty tests running a command on a remote pod with no tty
// GIVEN a pod in a cluster and a command to run on that pod
//
//	WHEN ExecPodNoTty is called
//	THEN ExecPodNoTty return the stdout, stderr, and a nil error
func TestExecPodNoTty(t *testing.T) {
	k8sutil.NewPodExecutor = spdyfake.NewPodExecutor
	spdyfake.PodExecResult = func(url *url.URL) (string, string, error) { return resultString, "", nil }
	cfg, client := spdyfake.NewClientsetConfig()
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "name",
		},
	}
	stdout, _, err := k8sutil.ExecPodNoTty(client, cfg, pod, "container", []string{"run", "some", "command"})
	assert.Nil(t, err)
	assert.Equal(t, resultString, stdout)
}

// TestGetURLForIngress tests getting the host URL from an ingress
// GIVEN an ingress name and its namespace
//
//	WHEN TestGetURLForIngress is called
//	THEN TestGetURLForIngress return the hostname if ingress exists, error otherwise
func TestGetURLForIngress(t *testing.T) {
	asserts := assert.New(t)
	ingress := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "test",
				},
			},
		},
	}
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&ingress).Build()
	ing, err := k8sutil.GetURLForIngress(client, "test", "default", "https")
	asserts.NoError(err)
	asserts.Equal("https://test", ing)

	client = fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	_, err = k8sutil.GetURLForIngress(client, "test", "default", "https")
	asserts.Error(err)
}

// TestGetRunningPodForLabel tests getting a running pod for a label
// GIVEN a running pod  with a label in a namespace in a cluster
//
//	WHEN GetRunningPodForLabel is called with that label and namespace
//	THEN GetRunningPodForLabel return the pod
func TestGetRunningPodForLabel(t *testing.T) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "name",
			Labels:    map[string]string{"key": "value"},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
	}
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(pod).Build()
	pod, err := k8sutil.GetRunningPodForLabel(client, "key=value", pod.GetNamespace())
	assert.Nil(t, err)
	assert.Equal(t, "name", pod.Name)
}

// TestGetCoreV1Client tests getting a CoreV1Client
//
//	WHEN GetCoreV1Client is called
//	THEN GetCoreV1Client returns a client and a nil error
func TestGetCoreV1Client(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)
	client, err := k8sutil.GetCoreV1Client()
	assert.Nil(t, err)
	assert.NotNil(t, client)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)

}

// TestGetAppsV1Client tests getting a AppsV1Client
//
//	WHEN GetAppsV1Client is called
//	THEN GetAppsV1Client returns a client and a nil error
func TestGetAppsV1Client(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)
	client, err := k8sutil.GetAppsV1Client()
	assert.Nil(t, err)
	assert.NotNil(t, client)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)

}

// TestGetKubernetesClientsetOrDie tests getting a KubernetesClientset
//
//	WHEN GetKubernetesClientsetOrDie is called
//	THEN GetKubernetesClientsetOrDie return clientset
func TestGetKubernetesClientsetOrDie(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)
	clientset := k8sutil.GetKubernetesClientsetOrDie()
	asserts.NotNil(clientset)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)

}

// TestGetDynamicClientInCluster tests getting a dynamic client
// GIVEN a kubeconfigpath
//
//	WHEN GetDynamicClientInCluster is called
//	THEN GetDynamicClientInCluster returns a client and a nil error
func TestGetDynamicClientInCluster(t *testing.T) {
	client, err := k8sutil.GetDynamicClientInCluster(dummyKubeConfig)
	assert.Nil(t, err)
	assert.NotNil(t, client)

}

// TestGetKubeConfigGivenPathAndContextWithNoKubeConfigPath tests getting a KubeConfig
// GIVEN a kubecontext but kubeConfigPath is missing
//
//	WHEN GetKubeConfigGivenPathAndContext is called
//	THEN GetKubeConfigGivenPathAndContext return the err and
func TestGetKubeConfigGivenPathAndContextWithNoKubeConfigPath(t *testing.T) {
	config, err := k8sutil.GetKubeConfigGivenPathAndContext("", "test")
	assert.Error(t, err)
	assert.Nil(t, config)

}

// TestErrorIfDeploymentExistsNoDeploy checks errors for deployments
// GIVEN a deployment doesn't exist
//
//	WHEN ErrorIfDeploymentExists is called
//	THEN ErrorIfDeploymentExists return a nil error
func TestErrorIfDeploymentExistsNoDeploy(t *testing.T) {
	k8sutil.GetAppsV1Func = common.MockGetAppsV1()
	err := k8sutil.ErrorIfDeploymentExists(appConfigNamespace, appConfigName)
	assert.Nil(t, err)
}

// TestErrorIfDeploymentExists checks errors for deployments
// GIVEN a deployment exist already
//
//	WHEN ErrorIfDeploymentExists is called
//	THEN ErrorIfDeploymentExists return an error
func TestErrorIfDeploymentExists(t *testing.T) {
	dep := common.MkDep(appConfigNamespace, appConfigName)
	k8sutil.GetAppsV1Func = common.MockGetAppsV1(dep)
	err := k8sutil.ErrorIfDeploymentExists(appConfigNamespace, appConfigName)
	assert.NotNil(t, err)
}

// TestErrorIfServiceExistsNoSvc checks errors for service
// GIVEN a service doesn't exist
//
//	WHEN ErrorIfServiceExists is called
//	THEN ErrorIfServiceExists returns a nil error
func TestErrorIfServiceExistsNoSvc(t *testing.T) {
	k8sutil.GetCoreV1Func = common.MockGetCoreV1()
	err := k8sutil.ErrorIfServiceExists(appConfigNamespace, appConfigName)
	assert.Nil(t, err)
}

// TestErrorIfServiceExists checks errors for service
// GIVEN a service exist already
//
//	WHEN ErrorIfServiceExists is called
//	THEN ErrorIfServiceExists returns an error
func TestErrorIfServiceExists(t *testing.T) {
	svc := common.MkSvc(appConfigNamespace, appConfigName)
	k8sutil.GetCoreV1Func = common.MockGetCoreV1(svc)
	err := k8sutil.ErrorIfServiceExists(appConfigNamespace, appConfigName)
	assert.NotNil(t, err)
}

// TestGetCertManagerClientset tests getting a cert manager clientset
//
//	WHEN GetCertManagerClienset is called
//	THEN GetCertManagerClienset return a non nil client and a nil error
func TestGetCertManagerClientset(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)
	client, err := k8sutil.GetCertManagerClienset()
	assert.Nil(t, err)
	assert.NotNil(t, client)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}
