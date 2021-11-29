// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8sutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"net/url"
	"os"
	"path/filepath"

	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioClient "istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// EnvVarKubeConfig Name of Environment Variable for KUBECONFIG
const EnvVarKubeConfig = "KUBECONFIG"

// EnvVarTestKubeConfig Name of Environment Variable for test KUBECONFIG
const EnvVarTestKubeConfig = "TEST_KUBECONFIG"

// GetKubeConfigLocation Helper function to obtain the default kubeConfig location
func GetKubeConfigLocation() (string, error) {
	if testKubeConfig := os.Getenv(EnvVarTestKubeConfig); len(testKubeConfig) > 0 {
		return testKubeConfig, nil
	}

	if kubeConfig := os.Getenv(EnvVarKubeConfig); len(kubeConfig) > 0 {
		return kubeConfig, nil
	}

	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config"), nil
	}

	return "", errors.New("unable to find kubeconfig")

}

// GetKubeConfigGivenPath GetKubeConfig will get the kubeconfig from the given kubeconfigPath
func GetKubeConfigGivenPath(kubeconfigPath string) (*restclient.Config, error) {
	return buildKubeConfig(kubeconfigPath)
}

func buildKubeConfig(kubeconfig string) (*restclient.Config, error) {
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

// GetKubeConfig Returns kubeconfig from KUBECONFIG env var if set
// Else from default location ~/.kube/config
func GetKubeConfig() (*rest.Config, error) {
	var config *rest.Config
	kubeConfigLoc, err := GetKubeConfigLocation()
	if err != nil {
		return config, err
	}
	config, err = clientcmd.BuildConfigFromFlags("", kubeConfigLoc)
	return config, err
}

// GetKubernetesClientset returns the Kubernetes clientset for the cluster set in the environment
func GetKubernetesClientset() (*kubernetes.Clientset, error) {
	// use the current context in the kubeconfig
	var clientset *kubernetes.Clientset
	config, err := GetKubeConfig()
	if err != nil {
		return clientset, err
	}
	clientset, err = kubernetes.NewForConfig(config)
	return clientset, err
}

// GetIstioClientset returns the clientset object for Istio
func GetIstioClientset() (*istioClient.Clientset, error) {
	kubeConfigLoc, err := GetKubeConfigLocation()
	if err != nil {
		return nil, err
	}
	return GetIstioClientsetInCluster(kubeConfigLoc)
}

// GetIstioClientsetInCluster returns the clientset object for Istio
func GetIstioClientsetInCluster(kubeconfigPath string) (*istioClient.Clientset, error) {
	var cs *istioClient.Clientset
	kubeConfig, err := GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		return cs, err
	}
	cs, err = istioClient.NewForConfig(kubeConfig)
	return cs, err
}

// GetHostnameFromGateway returns the host name from the application gateway that was
// created for the ApplicationConfiguration with name appConfigName from list of input gateways. If
// the input list of gateways is not provided, it is fetched from the kubernetes cluster
func GetHostnameFromGateway(namespace string, appConfigName string, gateways ...istiov1alpha3.Gateway) (string, error) {
	var config string
	kubeConfigLoc, err := GetKubeConfigLocation()
	if err != nil {
		return config, err
	}
	return GetHostnameFromGatewayInCluster(namespace, appConfigName, kubeConfigLoc, gateways...)
}

// GetHostnameFromGatewayInCluster returns the host name from the application gateway that was
// created for the ApplicationConfiguration with name appConfigName from list of input gateways. If
// the input list of gateways is not provided, it is fetched from the kubernetes cluster
func GetHostnameFromGatewayInCluster(namespace string, appConfigName string, kubeconfigPath string, gateways ...istiov1alpha3.Gateway) (string, error) {
	if len(gateways) == 0 {
		cs, err := GetIstioClientsetInCluster(kubeconfigPath)
		if err != nil {
			fmt.Printf("Could not get istio clientset: %v", err)
			return "", err
		}

		gatewayList, err := cs.NetworkingV1alpha3().Gateways(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Could not list application ingress gateways: %v", err)
			return "", err
		}

		gateways = gatewayList.Items
	}

	// if an optional appConfigName is provided, construct the gateway name from the namespace and
	// appConfigName and look for that specific gateway, otherwise just use the first gateway
	gatewayName := ""
	if len(appConfigName) > 0 {
		gatewayName = fmt.Sprintf("%s-%s-gw", namespace, appConfigName)
	}

	for _, gateway := range gateways {
		if len(gatewayName) > 0 && gatewayName != gateway.ObjectMeta.Name {
			continue
		}

		fmt.Printf("Found an app ingress gateway with name: %s\n", gateway.ObjectMeta.Name)
		if len(gateway.Spec.Servers) > 0 && len(gateway.Spec.Servers[0].Hosts) > 0 {
			return gateway.Spec.Servers[0].Hosts[0], nil
		}
	}

	// this can happen if the app gateway has not been created yet, the caller should
	// keep retrying and eventually we should get a gateway with a host
	fmt.Printf("Could not find host in application ingress gateways in namespace: %s\n", namespace)
	return "", nil
}

// NewPodExecutor is to be overridden during unit tests
var NewPodExecutor = remotecommand.NewSPDYExecutor

// FakePodSTDOUT can be used to output arbitrary strings during unit testing
var FakePodSTDOUT = ""

//NewFakePodExecutor should be used instead of remotecommand.NewSPDYExecutor in unit tests
func NewFakePodExecutor(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
	return &fakeExecutor{method: method, url: url}, nil
}

// fakeExecutor is for unit testing
type fakeExecutor struct {
	method string
	url    *url.URL
}

// Stream on a fakeExecutor sets stdout to FakePodSTDOUT
func (f *fakeExecutor) Stream(options remotecommand.StreamOptions) error {
	if options.Stdout != nil {
		buf := new(bytes.Buffer)
		buf.WriteString(FakePodSTDOUT)
		if _, err := options.Stdout.Write(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

//ExecPod runs a remote command a pod, returning the stdout and stderr of the command.
func ExecPod(cfg *rest.Config, restClient rest.Interface, pod *v1.Pod, container string, command []string) (string, string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	request := restClient.
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)
	executor, err := NewPodExecutor(cfg, "POST", request.URL())
	if err != nil {
		return "", "", err
	}
	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return "", "", fmt.Errorf("error running command %s on %v/%v: %v", command, pod.Namespace, pod.Name, err)
	}

	return stdout.String(), stderr.String(), nil
}
