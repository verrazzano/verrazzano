// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	oam "github.com/crossplane/oam-kubernetes-runtime/apis/core"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/semver"
	v1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vpoconstants "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	vzconstants "github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/github"
	adminv1 "k8s.io/api/admissionregistration/v1"
	appv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type VZHelper interface {
	GetOutputStream() io.Writer
	GetErrorStream() io.Writer
	GetInputStream() io.Reader
	GetClient(cmd *cobra.Command) (client.Client, error)
	GetKubeClient(cmd *cobra.Command) (kubernetes.Interface, error)
	GetHTTPClient() *http.Client
	GetDynamicClient(cmd *cobra.Command) (dynamic.Interface, error)
	GetDiscoveryClient(cmd *cobra.Command) (discovery.DiscoveryInterface, error)
	VerifyCLIArgsNil(cmd *cobra.Command) error
}

type ReportCtx struct {
	ReportFile           string
	ReportFormat         string
	IncludeSupportData   bool
	IncludeInfo          bool
	IncludeActions       bool
	MinConfidence        int
	MinImpact            int
	PrintReportToConsole bool
}

type ClusterSnapshotCtx struct {
	BugReportDir         string
	MoreNS               []string
	PrintReportToConsole bool
}

const defaultVerrazzanoTmpl = `apiVersion: install.verrazzano.io/%s
kind: Verrazzano
metadata:
  name: verrazzano
  namespace: default`

const v1beta1MinVersion = "1.4.0"

var (
	vzVer, k8sVer string
)

func NewVerrazzanoForVZVersion(version string) (schema.GroupVersion, client.Object, error) {
	if version == "" {
		// default to a v1beta1 compatible version if not specified
		version = "1.5.0"
	}
	actualVersion, err := semver.NewSemVersion(version)
	if err != nil {
		return schema.GroupVersion{}, nil, err
	}
	minVersion, err := semver.NewSemVersion(v1beta1MinVersion)
	if err != nil {
		return schema.GroupVersion{}, nil, err
	}
	if actualVersion.IsLessThan(minVersion) {
		o, err := newVerazzanoWithAPIVersion(v1alpha1.SchemeGroupVersion.Version)
		return v1alpha1.SchemeGroupVersion, o, err
	}
	o, err := newVerazzanoWithAPIVersion(v1beta1.SchemeGroupVersion.Version)
	return v1beta1.SchemeGroupVersion, o, err
}

func newVerazzanoWithAPIVersion(version string) (client.Object, error) {
	vz := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(fmt.Sprintf(defaultVerrazzanoTmpl, version)), vz); err != nil {
		return nil, err
	}
	return vz, nil
}

func NewVerrazzanoForGroupVersion(groupVersion schema.GroupVersion) func() interface{} {
	switch groupVersion {
	case v1alpha1.SchemeGroupVersion:
		return func() interface{} {
			return &v1alpha1.Verrazzano{}
		}
	default:
		return func() interface{} {
			return &v1beta1.Verrazzano{}
		}
	}
}

// FindVerrazzanoResource - find the single Verrazzano resource
func FindVerrazzanoResource(runtimeClient client.Client) (*v1beta1.Verrazzano, error) {
	vzList := v1beta1.VerrazzanoList{}
	err := runtimeClient.List(context.TODO(), &vzList)
	if err != nil {
		// If v1beta1 resource version doesn't exist, try v1alpha1
		if meta.IsNoMatchError(err) {
			return findVerazzanoResourceV1Alpha1(runtimeClient)
		}
		return nil, failedToFindResourceError(err)
	}
	if err := checkListLength(len(vzList.Items)); err != nil {
		return nil, err
	}
	return &vzList.Items[0], nil
}

// FindPlatformOperator - finds if the Verrazzano Platform Operator is already running
func FindPlatformOperator(runtimeClient client.Client) (bool, error) {
	var podList corev1.PodList
	err := runtimeClient.List(context.TODO(), &podList,
		client.InNamespace(constants.VerrazzanoInstallNamespace),
		client.MatchingLabels{"app": "verrazzano-platform-operator"})
	if err != nil {
		return false, err
	}
	if len(podList.Items) != 0 {
		return true, fmt.Errorf("A Verrazzano Platform Operator was found already running")
	}
	return false, nil
}

// GetVerrazzanoResource - get a Verrazzano resource
func GetVerrazzanoResource(client client.Client, namespacedName types.NamespacedName) (*v1beta1.Verrazzano, error) {
	vz := &v1beta1.Verrazzano{}
	if err := client.Get(context.TODO(), namespacedName, vz); err != nil {
		if meta.IsNoMatchError(err) {
			return getVerrazzanoResourceV1Alpha1(client, namespacedName)
		}
		return nil, failedToGetResourceError(err)

	}
	return vz, nil
}

func UpdateVerrazzanoResource(client client.Client, vz *v1beta1.Verrazzano) error {
	err := client.Update(context.TODO(), vz)
	// upgrade version may not support v1beta1
	if err != nil && (meta.IsNoMatchError(err) || apierrors.IsNotFound(err)) {
		vzV1Alpha1 := &v1alpha1.Verrazzano{}
		err = vzV1Alpha1.ConvertFrom(vz)
		if err != nil {
			return err
		}
		return client.Update(context.TODO(), vzV1Alpha1)
	}
	return err
}

// GetLatestReleaseVersion - get the version of the latest release of Verrazzano
func GetLatestReleaseVersion(client *http.Client) (string, error) {
	// Get the list of all Verrazzano releases
	releases, err := github.ListReleases(client)
	if err != nil {
		return "", fmt.Errorf("Failed to get list of Verrazzano releases: %s", err.Error())
	}
	return getLatestReleaseVersion(releases)
}

// getLatestReleaseVersion - determine which release it the latest based on semver values
func getLatestReleaseVersion(releases []string) (string, error) {
	var latestRelease *semver.SemVersion
	for _, tag := range releases {
		tagSemver, err := semver.NewSemVersion(tag)
		if err != nil {
			return "", err
		}
		if latestRelease == nil {
			// Initialize with the first tag
			latestRelease = tagSemver
		} else {
			if tagSemver.IsGreatherThan(latestRelease) {
				// Update the latest release found
				latestRelease = tagSemver
			}
		}
	}
	return fmt.Sprintf("v%s", latestRelease.ToString()), nil
}

func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
	_ = corev1.SchemeBuilder.AddToScheme(scheme)
	_ = adminv1.SchemeBuilder.AddToScheme(scheme)
	_ = rbacv1.SchemeBuilder.AddToScheme(scheme)
	_ = appv1.SchemeBuilder.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)
	_ = oam.AddToScheme(scheme)
	_ = batchv1.AddToScheme(scheme)
	_ = certv1.AddToScheme(scheme)
	return scheme
}

// GetNamespacesForAllComponents returns the list of unique namespaces of all the components included in the Verrazzano resource
func GetNamespacesForAllComponents(vz *v1beta1.Verrazzano) []string {
	var nsList []string
	if vz == nil {
		return nsList
	}
	allComponents := getAllComponents(vz)
	for _, eachComp := range allComponents {
		found, comp := registry.FindComponent(eachComp)
		if found {
			nsList = append(nsList, comp.Namespace())
		}
	}
	if len(nsList) > 0 {
		nsList = RemoveDuplicate(nsList)
	}
	return nsList
}

// getAllComponents returns the list of components from the Verrazzano resource
func getAllComponents(vzRes *v1beta1.Verrazzano) []string {
	var compSlice = make([]string, 0)

	for _, compStatusDetail := range vzRes.Status.Components {
		if compStatusDetail.State == v1beta1.CompStateDisabled {
			continue
		}
		compSlice = append(compSlice, compStatusDetail.Name)
	}
	return compSlice
}

func findVerazzanoResourceV1Alpha1(client client.Client) (*v1beta1.Verrazzano, error) {
	vzV1Alpha1List := v1alpha1.VerrazzanoList{}
	err := client.List(context.TODO(), &vzV1Alpha1List)
	if err != nil {
		return nil, failedToFindResourceError(err)
	}
	if err := checkListLength(len(vzV1Alpha1List.Items)); err != nil {
		return nil, err
	}
	vzConverted := &v1beta1.Verrazzano{}
	err = vzV1Alpha1List.Items[0].ConvertTo(vzConverted)
	return vzConverted, err
}

func getVerrazzanoResourceV1Alpha1(client client.Client, namespacedName types.NamespacedName) (*v1beta1.Verrazzano, error) {
	vzV1Alpha1 := &v1alpha1.Verrazzano{}
	if err := client.Get(context.TODO(), namespacedName, vzV1Alpha1); err != nil {
		return nil, failedToGetResourceError(err)
	}
	vz := &v1beta1.Verrazzano{}
	err := vzV1Alpha1.ConvertTo(vz)
	return vz, err
}

func failedToFindResourceError(err error) error {
	return fmt.Errorf("Failed to find any Verrazzano resources: %s", err.Error())
}

func failedToGetResourceError(err error) error {
	return fmt.Errorf("Failed to get a Verrazzano install resource: %s", err.Error())
}

func checkListLength(length int) error {
	if length == 0 {
		return fmt.Errorf("Failed to find any Verrazzano resources")
	}
	if length != 1 {
		return fmt.Errorf("Expected to only find one Verrazzano resource, but found %d", length)
	}
	return nil
}

// GetOperatorYaml returns Kubernetes manifests to deploy the Verrazzano platform operator
// The return value is operator.yaml for releases earlier than 1.4.0 and verrazzano-platform-operator.yaml from release 1.4.0 onwards
func GetOperatorYaml(version string) (string, error) {
	vzVersion, err := semver.NewSemVersion(version)
	if err != nil {
		return "", fmt.Errorf("invalid Verrazzano version: %v", err.Error())
	}
	ver140, _ := semver.NewSemVersion("v" + vpoconstants.VerrazzanoVersion1_4_0)
	var url string
	if vzVersion.IsGreaterThanOrEqualTo(ver140) {
		url = fmt.Sprintf(vzconstants.VerrazzanoPlatformOperatorURL, version)
	} else {
		url = fmt.Sprintf(vzconstants.VerrazzanoOperatorURL, version)
	}
	return url, nil
}

// SetK8sVer returns cluster Kubernetes version
func SetK8sVer() error {
	config, err := k8sutil.GetConfigFromController()
	if err != nil {
		return fmt.Errorf("error getting config from the Controller Runtime: %v", err.Error())
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error getting a clientset for the given config %v", err.Error())
	}

	versionInfo, err := client.ServerVersion()
	if err != nil {
		return fmt.Errorf("error getting kubernetes version %v", err.Error())
	}

	k8sVer = versionInfo.String()
	return nil
}

// SetVzVer set verrazzano version
func SetVzVer(client *client.Client) error {
	vz, vzErr := FindVerrazzanoResource(*client)
	if vzErr != nil {
		return vzErr
	}
	vzVer = vz.Status.Version
	return nil
}

// GetVersionOut returns the customised k8s and vz version string
func GetVersionOut() string {
	verOut := ""
	if vzVer != "" {
		verOut += fmt.Sprintf("\nVerrazzano Version: %s", vzVer)
	}
	if k8sVer != "" {
		verOut += fmt.Sprintf("\nKubernetes Version: %s\n", k8sVer)
	}
	return verOut
}

// VerifyVzInstallNamespaceExists returns existence of verrazzano-install namespace
func VerifyVzInstallNamespaceExists(kubeClient kubernetes.Interface) bool {
	pods, err := kubeClient.CoreV1().Pods(vzconstants.VerrazzanoInstall).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false
	}
	for i := range pods.Items {
		if strings.HasPrefix(pods.Items[i].Name, vzconstants.VerrazzanoPlatformOperator) {
			return true
		}
	}
	return false
}

// CheckAndRemoveBugReportExistsInDir checks vz bug report exists in dir or not
func CheckAndRemoveBugReportExistsInDir(dir string) bool {
	bugReportFilePattern := strings.Replace(vzconstants.BugReportFileDefaultValue, "-dt", "", 1)
	if fileMatched, _ := filepath.Glob(dir + bugReportFilePattern); len(fileMatched) == 1 {
		os.Remove(fileMatched[0])
		return true
	}
	return false
}
