// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"go.uber.org/zap"
	"io"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sYaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

var decUnstructured = k8sYaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

// GatherInfo invoked at the beginning to set up all the values taken as input
// The gingko runs will fail if any of these values are not set or set incorrectly
// The values are originally set from the jenkins pipeline
func GatherInfo() {
	VeleroNameSpace = os.Getenv("VELERO_NAMESPACE")
	VeleroOpenSearchSecretName = os.Getenv("VELERO_SECRET_NAME")
	VeleroMySQLSecretName = os.Getenv("VELERO_MYSQL_SECRET_NAME")
	RancherSecretName = os.Getenv("RANCHER_SECRET_NAME")
	OciBucketID = os.Getenv("OCI_OS_BUCKET_ID")
	OciBucketName = os.Getenv("OCI_OS_BUCKET_NAME")
	OciOsAccessKey = os.Getenv("OCI_OS_ACCESS_KEY")
	OciOsAccessSecretKey = os.Getenv("OCI_OS_ACCESS_SECRET_KEY")
	OciCompartmentID = os.Getenv("OCI_OS_COMPARTMENT_ID")
	OciNamespaceName = os.Getenv("OCI_OS_NAMESPACE")
	BackupResourceName = os.Getenv("BACKUP_RESOURCE")
	BackupOpensearchName = os.Getenv("BACKUP_OPENSEARCH")
	BackupRancherName = os.Getenv("BACKUP_RANCHER")
	BackupMySQLName = os.Getenv("BACKUP_MYSQL")
	RestoreOpensearchName = os.Getenv("RESTORE_OPENSEARCH")
	RestoreRancherName = os.Getenv("RESTORE_RANCHER")
	RestoreMySQLName = os.Getenv("RESTORE_MYSQL")
	BackupOpensearchStorageName = os.Getenv("BACKUP_OPENSEARCH_STORAGE")
	BackupMySQLStorageName = os.Getenv("BACKUP_MYSQL_STORAGE")
	BackupRegion = os.Getenv("BACKUP_REGION")
	OciCliTenancy = os.Getenv("OCI_CLI_TENANCY")
	OciCliUser = os.Getenv("OCI_CLI_USER")
	OciCliFingerprint = os.Getenv("OCI_CLI_FINGERPRINT")
	OciCliKeyFile = os.Getenv("OCI_CLI_KEY_FILE")
}

// GetRancherURL fetches the elastic search URL from the cluster
func GetRancherURL(log *zap.SugaredLogger) (string, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		log.Errorf("Failed to get kubeconfigPath with error: %v", err)
		return "", err
	}
	api, err := pkg.GetAPIEndpoint(kubeconfigPath)
	if err != nil {
		log.Errorf("Unable to fetch api endpoint due to %v", zap.Error(err))
		return "", err
	}
	ingress, err := api.GetIngress("cattle-system", "rancher")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host), nil
}

// GetRancherLoginToken fetches the login token for rancher console
func GetRancherLoginToken(log *zap.SugaredLogger) string {

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig url due to %v", zap.Error(err))
		return ""
	}

	httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		log.Errorf("Unable to fetch httpClient due to %v", zap.Error(err))
		return ""
	}

	rancherURL, err := GetRancherURL(log)
	if err != nil {
		return ""
	}

	return pkg.GetRancherAdminToken(log, httpClient, rancherURL)
}

// GetEsURL fetches the elastic search URL from the cluster
func GetEsURL(log *zap.SugaredLogger) (string, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		log.Errorf("Failed to get kubeconfigPath with error: %v", err)
		return "", err
	}
	api := pkg.EventuallyGetAPIEndpoint(kubeconfigPath)
	ingress, err := api.GetIngress(constants.VerrazzanoSystemNamespace, "vmi-system-es-ingest")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host), nil
}

// GetVZPasswd fetches the verrazzano password from the cluster
func GetVZPasswd(log *zap.SugaredLogger) (string, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return "", err
	}

	secret, err := clientset.CoreV1().Secrets(constants.VerrazzanoSystemNamespace).Get(context.TODO(), "verrazzano", metav1.GetOptions{})
	if err != nil {
		log.Infof("Error creating secret ", zap.Error(err))
		return "", err
	}
	return string(secret.Data["password"]), nil
}

// DynamicSSA uses dynamic client to apply data without registered golang structs
// This is used to apply configurations related to velero and rancher as they are crds
func DynamicSSA(ctx context.Context, deploymentYAML string, log *zap.SugaredLogger) error {

	kubeconfig, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Error getting kubeconfig, error: %v", err)
		return err
	}

	// Prepare a RESTMapper to find GVR followed by creating the dynamic client
	dc, err := discovery.NewDiscoveryClientForConfig(kubeconfig)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	dynamicClient, err := dynamic.NewForConfig(kubeconfig)
	if err != nil {
		return err
	}

	// Convert to unstructured since this will be used for CRDS
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode([]byte(deploymentYAML), nil, obj)
	if err != nil {
		return err
	}
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	// Create a dynamic REST interface
	var dynamicRest dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resources should specify the namespace
		dynamicRest = dynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		// for cluster-wide resources
		dynamicRest = dynamicClient.Resource(mapping.Resource)
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	//Apply the Yaml
	_, err = dynamicRest.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: "backup-controller",
	})
	return err
}

// CheckPvcsTerminated utility to wait for all pvcs to be terminated
func CheckPvcsTerminated(labelSelector, namespace string, log *zap.SugaredLogger) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	log.Infof("Wait for 60 seconds to allow pod termination as it has already been triggered")
	time.Sleep(60 * time.Second)

	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), listOptions)
	if err != nil {
		return err
	}
	if len(pvcs.Items) > 0 {
		log.Infof("Pvcs with label selector '%s' in namespace '%s' are still present", labelSelector, namespace)
		return fmt.Errorf("Pvcs with label selector '%s' in namespace '%s' are still present", labelSelector, namespace)
	}
	log.Infof("All pvcs with label selector '%s' in namespace '%s' have been removed", labelSelector, namespace)
	return nil

}

// DeleteSecret cleans up secrets as part of AfterSuite
func DeleteSecret(namespace string, name string, log *zap.SugaredLogger) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	err = clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		if !k8serror.IsNotFound(err) {
			log.Errorf("Failed to delete namespace '%s' due to: %v", name, zap.Error(err))
			return err
		}
	}
	return nil
}

// CreateCredentialsSecretFromFile creates opaque secret from a file
func CreateCredentialsSecretFromFile(namespace string, name string, log *zap.SugaredLogger) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	var b bytes.Buffer
	template, _ := template.New("testsecrets").Parse(SecretsData)
	data := AccessData{
		AccessName:             ObjectStoreCredsAccessKeyName,
		ScrtName:               ObjectStoreCredsSecretAccessKeyName,
		ObjectStoreAccessValue: OciOsAccessKey,
		ObjectStoreScrt:        OciOsAccessSecretKey,
	}

	template.Execute(&b, data)
	secretData := make(map[string]string)
	secretData["cloud"] = b.String()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: secretData,
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("Error creating secret ", zap.Error(err))
		return err
	}
	return nil
}

// CreateMySQLCredentialsSecretFromFile creates opaque secret from a file
func CreateMySQLCredentialsSecretFromFile(namespace string, name string, log *zap.SugaredLogger) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	var b bytes.Buffer
	template, err := template.New("testsecrets").Parse(SecretsData)
	if err != nil {
		return err
	}
	data := AccessData{
		AccessName:             ObjectStoreCredsAccessKeyName,
		ScrtName:               ObjectStoreCredsSecretAccessKeyName,
		ObjectStoreAccessValue: OciOsAccessKey,
		ObjectStoreScrt:        OciOsAccessSecretKey,
	}

	template.Execute(&b, data)

	profileTemplate, err := template.New("testprofile").Parse(ProfileData)
	if err != nil {
		return err
	}
	var profileByte bytes.Buffer
	pdata := InnoDBSecret{
		Region: BackupRegion,
	}
	profileTemplate.Execute(&profileByte, pdata)

	secretData := make(map[string]string)
	secretData["config"] = profileByte.String()
	secretData["credentials"] = b.String()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: secretData,
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("Error creating secret ", zap.Error(err))
		return err
	}
	return nil
}

// CreateMySQLCredentialsSecretFromFile creates opaque secret from a file
func CreateMySQLCredentialsSecretFromUserPrincipal(namespace string, name string, log *zap.SugaredLogger) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	keydata, err := os.ReadFile(OciCliKeyFile)
	if err != nil {
		log.Errorf("Unable to read file '%s' due to %v", OciCliKeyFile, zap.Error(err))
	}

	secretData := make(map[string]string)
	secretData["region"] = BackupRegion
	secretData["passphrase"] = ""
	secretData["user"] = OciCliUser
	secretData["tenancy"] = OciCliTenancy
	secretData["fingerprint"] = OciCliFingerprint
	secretData["privatekey"] = string(keydata)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: secretData,
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("Error creating secret ", zap.Error(err))
		return err
	}
	return nil
}

// DeleteNamespace method to delete a namespace
func DeleteNamespace(namespace string, log *zap.SugaredLogger) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	err = clientset.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
	if err != nil {
		if !k8serror.IsNotFound(err) {
			log.Errorf("Failed to delete namespace '%s' due to: %v", namespace, zap.Error(err))
			return err
		}
	}
	return nil
}

// HTTPHelper utility for http method use cases
func HTTPHelper(httpClient *retryablehttp.Client, method, httpURL, token, tokenType string, expectedResponseCode int, payload interface{}, log *zap.SugaredLogger) (*gabs.Container, error) {

	var retryabeRequest *retryablehttp.Request
	var err error

	switch method {
	case "GET":
		retryabeRequest, err = retryablehttp.NewRequest(http.MethodGet, httpURL, payload)
		retryabeRequest.Header.Set("Content-Type", "application/json")
	case "POST":
		retryabeRequest, err = retryablehttp.NewRequest(http.MethodPost, httpURL, payload)
	case "DELETE":
		retryabeRequest, err = retryablehttp.NewRequest(http.MethodDelete, httpURL, payload)
	}
	if err != nil {
		log.Error(fmt.Sprintf("error creating retryable api request for %s: %v", httpURL, err))
		return nil, err
	}

	switch tokenType {
	case "Bearer":
		retryabeRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	case "Basic":
		retryabeRequest.SetBasicAuth(strings.Split(token, ":")[0], strings.Split(token, ":")[1])
	}
	retryabeRequest.Header.Set("Accept", "application/json")
	response, err := httpClient.Do(retryabeRequest)
	if err != nil {
		log.Error(fmt.Sprintf("error invoking api request %s: %v", httpURL, err))
		return nil, err
	}
	defer response.Body.Close()

	err = httputil.ValidateResponseCode(response, expectedResponseCode)
	if err != nil {
		log.Errorf("expected response code = %v, actual response code = %v, Error = %v", expectedResponseCode, response.StatusCode, zap.Error(err))
		return nil, err
	}

	// extract the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Failed to read response body: %v", zap.Error(err))
		return nil, err
	}

	jsonParsed, err := gabs.ParseJSON(body)
	if err != nil {
		log.Errorf("Failed to parse json: %v", zap.Error(err))
		return nil, err
	}

	return jsonParsed, nil
}

// DisplayHookLogs is used to display the logs from the pod where the backup hook was run
// It execs into the pod and fetches the log file contents
func DisplayHookLogs(log *zap.SugaredLogger) error {

	log.Infof("Retrieving verrazzano hook logs ...")
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Failed to get config with error: %v", err)
		return err
	}

	podSpec, err := clientset.CoreV1().Pods(constants.VerrazzanoSystemNamespace).Get(context.TODO(), "vmi-system-es-master-0", metav1.GetOptions{})
	if err != nil {
		return err
	}

	cmdLogFileName := []string{"/bin/sh", "-c", "ls -alt --time=ctime /tmp/ | grep verrazzano | head -1"}
	stdout, _, err := k8sutil.ExecPod(clientset, config, podSpec, "es-master", cmdLogFileName)
	if err != nil {
		log.Errorf("Error = %v", zap.Error(err))
		return err
	}

	logFileData := strings.TrimSpace(strings.Trim(stdout, "\n"))
	logFileName := strings.Split(logFileData, " ")[len(strings.Split(logFileData, " "))-1]

	var execCmd []string
	execCmd = append(execCmd, "cat")
	execCmd = append(execCmd, fmt.Sprintf("/tmp/%s", logFileName))
	stdout, _, err = k8sutil.ExecPod(clientset, config, podSpec, "es-master", execCmd)
	if err != nil {
		log.Errorf("Error = %v", zap.Error(err))
		return err
	}
	log.Infof(stdout)
	return nil
}

func Runner(bcmd *BashCommand, log *zap.SugaredLogger) *RunnerResponse {
	var stdoutBuf, stderrBuf bytes.Buffer
	var bashCommandResponse RunnerResponse
	bashCommand := exec.Command(bcmd.CommandArgs[0], bcmd.CommandArgs[1:]...) //nolint:gosec
	bashCommand.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	bashCommand.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	log.Infof("Executing command '%v'", bashCommand.String())
	err := bashCommand.Run()
	if err != nil {
		log.Errorf("Cmd '%v' execution failed due to '%v'", bashCommand.String(), zap.Error(err))
		bashCommandResponse.CommandError = err
		return &bashCommandResponse
	}
	bashCommandResponse.StandardOut = stdoutBuf
	bashCommandResponse.CommandError = err
	return &bashCommandResponse
}
