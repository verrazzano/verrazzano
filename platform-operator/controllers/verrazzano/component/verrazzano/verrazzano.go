// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/namespace"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"io/fs"
	"io/ioutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	neturl "net/url"
	"os"
	"os/exec"
	controllerruntime "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"strings"
	"text/template"
	"time"
)

// ComponentName is the name of the component

const (
	ComponentName           = "verrazzano"
	keycloakInClusterURL    = "keycloak-http.keycloak.svc.cluster.local"
	esHelmValuePrefixFormat = "elasticSearch.%s"

	workloadName  = "system-es-master"
	containerName = "es-master"
	portName      = "http"
	indexPattern  = "verrazzano-*"
	notFound      = 404
	searchPort    = "9200"

	tmpFilePrefix        = "verrazzano-overrides-"
	tmpSuffix            = "yaml"
	tmpFileCreatePattern = tmpFilePrefix + "*." + tmpSuffix
	tmpFileCleanPattern  = tmpFilePrefix + ".*\\." + tmpSuffix
)

const indexTemplatePayload = `{
    "index_patterns":[
		"verrazzano-system",
		"verrazzano-application*"
    ],
    "version":60001,
    "priority": 100,
    "data_stream": {},
    "template": {
        "settings":{
            "index.refresh_interval":"5s",
            "index.mapping.total_fields.limit":"2000",
            "number_of_shards":5,
            "index.number_of_replicas":0,
            "index.auto_expand_replicas":"0-1"
        },
        "mappings":{
            "dynamic_templates":[
            {
                "message_field":{
                "path_match":"message",
                "match_mapping_type":"string",
                "mapping":{
                    "type":"text",
                    "norms":false
                }
                }
            },
            {
                "string_fields":{
                "match":"*",
                "match_mapping_type":"string",
                "mapping":{
                    "type":"text",
                    "norms":false,
                    "fields":{
                    "keyword":{
                        "type":"keyword",
                        "ignore_above":256
                    }
                    }
                }
                }
            }
            ]
        }
    }
}`

const (
	minIndexAge        = "min_index_age"
	defaultMinIndexAge = "7d"
)

const systemISMPayloadTemplate = `{
    "policy": {
        "policy_id": "system_ingest_delete",
        "description": "Verrazzano Index policy to rollover and delete system indices",
        "schema_version": 12,
        "error_notification": null,
        "default_state": "ingest",
        "states": [
            {
                "name": "ingest",
                "actions": [
                    {
                        "rollover": {
                            "min_index_age": "1d"
                        }
                    }
                ],
                "transitions": [
                    {
                        "state_name": "delete",
                        "conditions": {
                            "min_index_age": "{{ .min_index_age }}"
                        }
                    }
                ]
            },
            {
                "name": "delete",
                "actions": [
                    {
                        "delete": {}
                    }
                ],
                "transitions": []
            }
        ],
        "ism_template": {
          "index_patterns": [
            "verrazzano-system"
          ],
          "priority": 1
        }
    }
}`

const applicationISMPayloadTemplate = `{
    "policy": {
        "policy_id": "application_ingest_delete",
        "description": "Verrazzano Index policy to rollover and delete application indices",
        "schema_version": 12,
        "error_notification": null,
        "default_state": "ingest",
        "states": [
            {
                "name": "ingest",
                "actions": [
                    {
                        "rollover": {
                            "min_index_age": "1d"
                        }
                    }
                ],
                "transitions": [
                    {
                        "state_name": "delete",
                        "conditions": {
                            "min_index_age": "{{ .min_index_age }}"
                        }
                    }
                ]
            },
            {
                "name": "delete",
                "actions": [
                    {
                        "delete": {}
                    }
                ],
                "transitions": []
            }
        ],
        "ism_template": {
          "index_patterns": [
            "verrazzano-application*"
          ],
          "priority": 1
        }
    }
}`

type (
	ISMPolicy struct {
		PrimaryTerm    int `json:"_primary_term"`
		SequenceNumber int `json:"_seq_no"`
		Status         int `json:"status"`
	}
	uriComponents struct {
		host   string
		port   string
		scheme string
	}
)

var (
	// For Unit test purposes
	execCommand   = exec.Command
	writeFileFunc = ioutil.WriteFile
)

func resetWriteFileFunc() {
	writeFileFunc = ioutil.WriteFile
}

// resolveVerrazzanoNamespace will return the default Verrazzano system namespace unless the namespace is specified
func resolveVerrazzanoNamespace(ns string) string {
	if len(ns) > 0 && ns != "default" {
		return ns
	}
	return globalconst.VerrazzanoSystemNamespace
}

// VerrazzanoPreUpgrade contains code that is run prior to helm upgrade for the Verrazzano helm chart
func verrazzanoPreUpgrade(log vzlog.VerrazzanoLogger, client clipkg.Client, _ string, namespace string, _ string) error {
	return fixupFluentdDaemonset(log, client, namespace)
}

// appendVerrazzanoOverrides appends the image overrides for the monitoring-init-images subcomponent
func appendVerrazzanoOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {

	// Append some custom image overrides
	// - use local KeyValues array to ensure we append those after the file override; typically won't matter with the
	//   way we implement Helm calls, but don't depend on that
	vzkvs, err := appendCustomImageOverrides(kvs)
	if err != nil {
		return kvs, err
	}

	effectiveCR := ctx.EffectiveCR()
	// Find any storage overrides for the VMI, and
	resourceRequestOverrides, err := findStorageOverride(effectiveCR)
	if err != nil {
		return kvs, err
	}

	// Overrides object to store any user overrides
	overrides := verrazzanoValues{}

	// Append the simple overrides
	if err := appendVerrazzanoValues(ctx, &overrides); err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}
	// Append any VMI overrides to the override values object, and any installArgs overrides to the kvs list
	vzkvs = appendVMIOverrides(effectiveCR, &overrides, resourceRequestOverrides, vzkvs)

	// append any fluentd overrides
	if err := appendFluentdOverrides(effectiveCR, &overrides); err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}
	// append the security role overrides
	if err := appendSecurityOverrides(effectiveCR, &overrides); err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	// Append any installArgs overrides to the kvs list
	vzkvs = appendVerrazzanoComponentOverrides(effectiveCR, vzkvs)

	// Write the overrides file to a temp dir and add a helm file override argument
	overridesFileName, err := generateOverridesFile(ctx, &overrides)
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	// Append any installArgs overrides in vzkvs after the file overrides to ensure precedence of those
	kvs = append(kvs, bom.KeyValue{Value: overridesFileName, IsFile: true})
	kvs = append(kvs, vzkvs...)
	return kvs, nil
}

func appendCustomImageOverrides(kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	imageOverrides, err := bomFile.BuildImageOverrides("monitoring-init-images")
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	kvs = append(kvs, imageOverrides...)
	return kvs, nil
}

func generateOverridesFile(ctx spi.ComponentContext, overrides *verrazzanoValues) (string, error) {
	bytes, err := yaml.Marshal(overrides)
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp(os.TempDir(), tmpFileCreatePattern)
	if err != nil {
		return "", err
	}

	overridesFileName := file.Name()
	if err := writeFileFunc(overridesFileName, bytes, fs.ModeAppend); err != nil {
		return "", err
	}
	ctx.Log().Debugf("Verrazzano install overrides file %s contents: %s", overridesFileName, string(bytes))
	return overridesFileName, nil
}

func appendVerrazzanoValues(ctx spi.ComponentContext, overrides *verrazzanoValues) error {
	effectiveCR := ctx.EffectiveCR()
	if isWildcardDNS, domain := getWildcardDNS(&effectiveCR.Spec); isWildcardDNS {
		overrides.DNS = &dnsValues{
			Wildcard: &wildcardDNSSettings{
				Domain: domain,
			},
		}
	}

	dnsSuffix, err := vzconfig.GetDNSSuffix(ctx.Client(), effectiveCR)
	if err != nil {
		return ctrlerrors.RetryableError{
			Source: ComponentName,
			Cause:  err,
		}
	}

	if externalDNSEnabled := vzconfig.IsExternalDNSEnabled(effectiveCR); externalDNSEnabled {
		overrides.Externaldns = &externalDNSValues{
			Enabled: externalDNSEnabled,
		}
	}

	envName := vzconfig.GetEnvName(effectiveCR)
	overrides.Config = &configValues{
		EnvName:   envName,
		DNSSuffix: dnsSuffix,
	}

	overrides.Keycloak = &keycloakValues{Enabled: vzconfig.IsKeycloakEnabled(effectiveCR)}
	overrides.Rancher = &rancherValues{Enabled: vzconfig.IsRancherEnabled(effectiveCR)}
	overrides.Console = &consoleValues{Enabled: vzconfig.IsConsoleEnabled(effectiveCR)}
	overrides.VerrazzanoOperator = &voValues{Enabled: isVMOEnabled(effectiveCR)}
	overrides.MonitoringOperator = &vmoValues{Enabled: isVMOEnabled(effectiveCR)}
	overrides.API = &apiValues{
		Proxy: &proxySettings{
			OidcProviderHost:          fmt.Sprintf("keycloak.%s.%s", envName, dnsSuffix),
			OidcProviderHostInCluster: keycloakInClusterURL,
		},
	}
	return nil
}

func appendSecurityOverrides(effectiveCR *vzapi.Verrazzano, overrides *verrazzanoValues) error {
	vzSpec := effectiveCR.Spec

	numAdminSubjects := len(vzSpec.Security.AdminSubjects)
	numMonSubjects := len(vzSpec.Security.MonitorSubjects)
	if numMonSubjects == 0 && numAdminSubjects == 0 {
		return nil
	}

	overrides.Security = &securityRoleBindingValues{}

	if numAdminSubjects > 0 {
		adminSubjectsMap := make(map[string]subject, numAdminSubjects)
		for i, adminSubj := range vzSpec.Security.AdminSubjects {
			subjectName := fmt.Sprintf("subject-%d", i)
			if err := vzconfig.ValidateRoleBindingSubject(adminSubj, subjectName); err != nil {
				return err
			}
			adminSubjectsMap[subjectName] = subject{
				Name:      adminSubj.Name,
				Kind:      adminSubj.Kind,
				Namespace: adminSubj.Namespace,
				APIGroup:  adminSubj.APIGroup,
			}
		}
		overrides.Security.AdminSubjects = adminSubjectsMap
	}
	if numMonSubjects > 0 {
		monSubjectMap := make(map[string]subject, numMonSubjects)
		for i, monSubj := range vzSpec.Security.MonitorSubjects {
			subjectName := fmt.Sprintf("subject-%d", i)
			if err := vzconfig.ValidateRoleBindingSubject(monSubj, fmt.Sprintf("monitorSubjects[%d]", i)); err != nil {
				return err
			}
			monSubjectMap[subjectName] = subject{
				Name:      monSubj.Name,
				Kind:      monSubj.Kind,
				Namespace: monSubj.Namespace,
				APIGroup:  monSubj.APIGroup,
			}
		}
		overrides.Security.MonitorSubjects = monSubjectMap
	}
	return nil
}

// appendVerrazzanoComponentOverrides - append overrides specified for the Verrazzano component
func appendVerrazzanoComponentOverrides(effectiveCR *vzapi.Verrazzano, kvs []bom.KeyValue) []bom.KeyValue {
	if effectiveCR.Spec.Components.Verrazzano != nil {
		for _, arg := range effectiveCR.Spec.Components.Verrazzano.InstallArgs {
			kvs = append(kvs, bom.KeyValue{
				Key:   arg.Name,
				Value: arg.Value,
			})
		}
	}
	return kvs
}

func appendVMIOverrides(effectiveCR *vzapi.Verrazzano, overrides *verrazzanoValues, storageOverrides *resourceRequestValues, kvs []bom.KeyValue) []bom.KeyValue {
	overrides.Kibana = &kibanaValues{Enabled: vzconfig.IsKibanaEnabled(effectiveCR)}

	overrides.ElasticSearch = &elasticsearchValues{
		Enabled: vzconfig.IsElasticsearchEnabled(effectiveCR),
	}
	if storageOverrides != nil {
		overrides.ElasticSearch.Nodes = &esNodes{
			// Only have to override the data node storage
			Data: &esNodeValues{
				Requests: storageOverrides,
			},
		}
	}
	if effectiveCR.Spec.Components.Elasticsearch != nil {
		for _, arg := range effectiveCR.Spec.Components.Elasticsearch.ESInstallArgs {
			kvs = append(kvs, bom.KeyValue{
				Key:   fmt.Sprintf(esHelmValuePrefixFormat, arg.Name),
				Value: arg.Value,
			})
		}
	}

	overrides.Prometheus = &prometheusValues{
		Enabled:  vzconfig.IsPrometheusEnabled(effectiveCR),
		Requests: storageOverrides,
	}

	overrides.Grafana = &grafanaValues{
		Enabled:  vzconfig.IsGrafanaEnabled(effectiveCR),
		Requests: storageOverrides,
	}
	return kvs
}

func findStorageOverride(effectiveCR *vzapi.Verrazzano) (*resourceRequestValues, error) {
	if effectiveCR == nil || effectiveCR.Spec.DefaultVolumeSource == nil {
		return nil, nil
	}
	defaultVolumeSource := effectiveCR.Spec.DefaultVolumeSource
	if defaultVolumeSource.EmptyDir != nil {
		return &resourceRequestValues{
			Storage: "",
		}, nil
	}
	if defaultVolumeSource.PersistentVolumeClaim != nil {
		pvcClaim := defaultVolumeSource.PersistentVolumeClaim
		storageSpec, found := vzconfig.FindVolumeTemplate(pvcClaim.ClaimName, effectiveCR.Spec.VolumeClaimSpecTemplates)
		if !found {
			return nil, fmt.Errorf("Did not find matching storage volume template for claim %s", pvcClaim.ClaimName)
		}
		storageString := storageSpec.Resources.Requests.Storage().String()
		return &resourceRequestValues{
			Storage: storageString,
		}, nil
	}
	return nil, fmt.Errorf("Unsupported volume source: %v", defaultVolumeSource)
}

func newURIComponents(url string) (*uriComponents, error) {
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return nil, err
	}
	uriComp := &uriComponents{}
	if len(parsedURL.Port()) > 0 {
		uriComp.port = parsedURL.Port()
	} else {
		uriComp.port = searchPort
	}
	uriComp.scheme = parsedURL.Scheme
	uriComp.host = strings.Split(parsedURL.Host, ":")[0]
	return uriComp, nil
}

func setLoggingOverrides(values *loggingValues, url string) error {
	uriComp, err := newURIComponents(url)
	if err != nil {
		return err
	}
	values.ElasticsearchURL = uriComp.host
	values.ElasticsearchPort = uriComp.port
	values.ElasticsearchScheme = uriComp.scheme
	return nil
}

func appendFluentdOverrides(effectiveCR *vzapi.Verrazzano, overrides *verrazzanoValues) error {
	overrides.Fluentd = &fluentdValues{
		Enabled: vzconfig.IsFluentdEnabled(effectiveCR),
	}

	fluentd := effectiveCR.Spec.Components.Fluentd
	if fluentd != nil {
		overrides.Logging = &loggingValues{}
		if len(fluentd.ElasticsearchURL) > 0 {
			if err := setLoggingOverrides(overrides.Logging, fluentd.ElasticsearchURL); err != nil {
				return err
			}
		}
		if len(fluentd.ElasticsearchSecret) > 0 {
			overrides.Logging.ElasticsearchSecret = fluentd.ElasticsearchSecret
		}
		if len(fluentd.ExtraVolumeMounts) > 0 {
			for _, vm := range fluentd.ExtraVolumeMounts {
				dest := vm.Source
				if vm.Destination != "" {
					dest = vm.Destination
				}
				readOnly := true
				if vm.ReadOnly != nil {
					readOnly = *vm.ReadOnly
				}
				overrides.Fluentd.ExtraVolumeMounts = append(overrides.Fluentd.ExtraVolumeMounts,
					volumeMount{Source: vm.Source, Destination: dest, ReadOnly: readOnly})
			}
		}
		// Overrides for OCI Logging integration
		if fluentd.OCI != nil {
			overrides.Fluentd.OCI = &ociLoggingSettings{
				DefaultAppLogID: fluentd.OCI.DefaultAppLogID,
				SystemLogID:     fluentd.OCI.SystemLogID,
				APISecret:       fluentd.OCI.APISecret,
			}
		}
	}

	return nil
}

func isVMOEnabled(vz *vzapi.Verrazzano) bool {
	return vzconfig.IsPrometheusEnabled(vz) || vzconfig.IsKibanaEnabled(vz) || vzconfig.IsElasticsearchEnabled(vz) || vzconfig.IsGrafanaEnabled(vz)
}

func getWildcardDNS(vz *vzapi.VerrazzanoSpec) (bool, string) {
	if vz.Components.DNS != nil && vz.Components.DNS.Wildcard != nil {
		return true, vz.Components.DNS.Wildcard.Domain
	}
	return false, ""
}

// This function is used to fixup the fluentd daemonset on a managed cluster so that helm upgrade of Verrazzano does
// not fail.  Prior to Verrazzano v1.0.1, the mcagent would change the environment variables CLUSTER_NAME and
// ELASTICSEARCH_URL on a managed cluster to use valueFrom (from a secret) instead of using a Value. The helm chart
// template for the fluentd daemonset expects a Value.
func fixupFluentdDaemonset(log vzlog.VerrazzanoLogger, client clipkg.Client, namespace string) error {
	// Get the fluentd daemonset resource
	fluentdNamespacedName := types.NamespacedName{Name: globalconst.FluentdDaemonSetName, Namespace: namespace}
	daemonSet := appsv1.DaemonSet{}
	err := client.Get(context.TODO(), fluentdNamespacedName, &daemonSet)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		log.Errorf("Failed to find the fluentd DaemonSet %s, %v", daemonSet.Name, err)
		return err
	}

	// Find the fluent container and save it's container index
	fluentdIndex := -1
	for i, container := range daemonSet.Spec.Template.Spec.Containers {
		if container.Name == "fluentd" {
			fluentdIndex = i
			break
		}
	}
	if fluentdIndex == -1 {
		return fmt.Errorf("fluentd container not found in fluentd daemonset: %s", daemonSet.Name)
	}

	// Check if env variables CLUSTER_NAME and ELASTICSEARCH_URL are using valueFrom.
	clusterNameIndex := -1
	elasticURLIndex := -1
	for i, env := range daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env {
		if env.Name == constants.ClusterNameEnvVar && env.ValueFrom != nil {
			clusterNameIndex = i
			continue
		}
		if env.Name == constants.ElasticsearchURLEnvVar && env.ValueFrom != nil {
			elasticURLIndex = i
		}
	}

	// If valueFrom is not being used then we do not need to fix the env variables
	if clusterNameIndex == -1 && elasticURLIndex == -1 {
		return nil
	}

	// Get the secret containing managed cluster name and Elasticsearch URL
	secretNamespacedName := types.NamespacedName{Name: constants.MCRegistrationSecret, Namespace: namespace}
	secret := corev1.Secret{}
	err = client.Get(context.TODO(), secretNamespacedName, &secret)
	if err != nil {
		return err
	}

	// The secret must contain a cluster name
	clusterName, ok := secret.Data[constants.ClusterNameData]
	if !ok {
		return fmt.Errorf("the secret named %s in namespace %s is missing the required field %s", secret.Name, secret.Namespace, constants.ClusterNameData)
	}

	// The secret must contain the Elasticsearch endpoint's URL
	elasticsearchURL, ok := secret.Data[constants.ElasticsearchURLData]
	if !ok {
		return fmt.Errorf("the secret named %s in namespace %s is missing the required field %s", secret.Name, secret.Namespace, constants.ElasticsearchURLData)
	}

	// Update the daemonset to use a Value instead of the valueFrom
	if clusterNameIndex != -1 {
		daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env[clusterNameIndex].Value = string(clusterName)
		daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env[clusterNameIndex].ValueFrom = nil
	}
	if elasticURLIndex != -1 {
		daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env[elasticURLIndex].Value = string(elasticsearchURL)
		daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env[elasticURLIndex].ValueFrom = nil
	}
	log.Debug("Updating fluentd daemonset to use valueFrom instead of Value for CLUSTER_NAME and ELASTICSEARCH_URL environment variables")
	err = client.Update(context.TODO(), &daemonSet)
	return err
}

func createAndLabelNamespaces(ctx spi.ComponentContext) error {
	if err := LabelKubeSystemNamespace(ctx.Client()); err != nil {
		return err
	}
	if err := namespace.CreateVerrazzanoSystemNamespace(ctx.Client()); err != nil {
		return err
	}
	if _, err := secret.CheckImagePullSecret(ctx.Client(), globalconst.VerrazzanoSystemNamespace); err != nil {
		return ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}
	if err := namespace.CreateVerrazzanoMultiClusterNamespace(ctx.Client()); err != nil {
		return err
	}
	if isVMOEnabled(ctx.EffectiveCR()) {
		// If the monitoring operator is enabled, create the monitoring namespace and copy the image pull secret
		if err := namespace.CreateVerrazzanoMonitoringNamespace(ctx.Client()); err != nil {
			return err
		}
		if _, err := secret.CheckImagePullSecret(ctx.Client(), globalconst.VerrazzanoMonitoringNamespace); err != nil {
			return ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
		}
	}
	if vzconfig.IsKeycloakEnabled(ctx.EffectiveCR()) {
		if err := namespace.CreateKeycloakNamespace(ctx.Client()); err != nil {
			return ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
		}
	}
	if vzconfig.IsRancherEnabled(ctx.EffectiveCR()) {
		if err := namespace.CreateAndLabelNamespace(ctx.Client(), globalconst.RancherOperatorSystemNamespace,
			true, false); err != nil {
			return ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
		}
	}
	// cattle-system NS must be created since the rancher NetworkPolicy, which is always installed, requires it
	if err := namespace.CreateRancherNamespace(ctx.Client()); err != nil {
		return ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}
	return nil
}

// LabelKubeSystemNamespace adds the label needed by network polices to kube-system
func LabelKubeSystemNamespace(client clipkg.Client) error {
	const KubeSystemNamespace = "kube-system"
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: KubeSystemNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), client, &ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["verrazzano.io/namespace"] = KubeSystemNamespace
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// loggingPreInstall copies logging secrets from the verrazzano-install namespace to the verrazzano-system namespace
func loggingPreInstall(ctx spi.ComponentContext) error {
	if vzconfig.IsFluentdEnabled(ctx.EffectiveCR()) {
		// If fluentd is enabled, copy any custom secrets
		fluentdConfig := ctx.EffectiveCR().Spec.Components.Fluentd
		if fluentdConfig != nil {
			// Copy the external Elasticsearch secret
			if len(fluentdConfig.ElasticsearchURL) > 0 && fluentdConfig.ElasticsearchSecret != globalconst.DefaultElasticsearchSecretName {
				if err := copySecret(ctx, fluentdConfig.ElasticsearchSecret, "custom Elasticsearch"); err != nil {
					return err
				}
			}
			// Copy the OCI API secret
			if fluentdConfig.OCI != nil && len(fluentdConfig.OCI.APISecret) > 0 {
				if err := copySecret(ctx, fluentdConfig.OCI.APISecret, "OCI API"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// copySecret copies a secret from the verrazzano-install namespace to the verrazzano-system namespace. If
// the target secret already exists, then it will be updated if necessary.
func copySecret(ctx spi.ComponentContext, secretName string, logMsg string) error {
	vzLog := ctx.Log()
	vzLog.Debugf("Copying %s secret %s to %s namespace", logMsg, secretName, globalconst.VerrazzanoSystemNamespace)

	targetSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
	}
	opResult, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &targetSecret, func() error {
		sourceSecret := corev1.Secret{}
		nsn := types.NamespacedName{Name: secretName, Namespace: constants.VerrazzanoInstallNamespace}
		if err := ctx.Client().Get(context.TODO(), nsn, &sourceSecret); err != nil {
			return err
		}
		targetSecret.Type = sourceSecret.Type
		targetSecret.Immutable = sourceSecret.Immutable
		targetSecret.StringData = sourceSecret.StringData
		targetSecret.Data = sourceSecret.Data
		return nil
	})

	vzLog.Debugf("Copy %s secret result: %s", logMsg, opResult)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrlerrors.RetryableError{
				Source: ComponentName,
				Cause:  err,
			}
		}
		vzLog.Errorf("The %s secret %s not found in namespace %s", logMsg, secretName, constants.VerrazzanoInstallNamespace)
		return ctrlerrors.RetryableError{Source: ComponentName}
	}

	return nil
}

// isVerrazzanoSecretReady returns true if the Verrazzano secret is present in the system namespace
func isVerrazzanoSecretReady(ctx spi.ComponentContext) bool {
	if err := ctx.Client().Get(context.TODO(),
		types.NamespacedName{Name: "verrazzano", Namespace: globalconst.VerrazzanoSystemNamespace},
		&corev1.Secret{}); err != nil {
		if !errors.IsNotFound(err) {
			ctx.Log().Error("Unexpected error getting verrazzano secret: %s", err)
			return false
		}
		ctx.Log().Debugf("Verrazzano secret not found")
		return false
	}
	return true
}

//cleanTempFiles - Clean up the override temp files in the temp dir
func cleanTempFiles(ctx spi.ComponentContext) {
	if err := vzos.RemoveTempFiles(ctx.Log().GetZapLogger(), tmpFileCleanPattern); err != nil {
		ctx.Log().Errorf("Error deleting temp files: %s", err.Error())
	}
}

func isOpenSearchReady(ctx spi.ComponentContext, namespace string) ([]corev1.Pod, bool) {
	pods, err := getPodsWithReadyContainer(ctx.Client(), containerName, clipkg.MatchingLabels{"app": workloadName}, clipkg.InNamespace(namespace))
	if err != nil {
		return nil, false
	}
	if len(pods) < 1 {
		return nil, false
	}
	return pods, true
}

func makeBashCommand(command string) []string {
	return []string{
		"bash",
		"-c",
		command,
	}
}

func doPUT(cfg *rest.Config, cli kubernetes.Interface, pod *corev1.Pod, cmd string) error {
	command := "curl -X PUT -H 'Content-Type: application/json' localhost:9200/" + cmd
	_, _, err := k8sutil.ExecPod(cli, cfg, pod, containerName, makeBashCommand(command))
	return err
}

var ConfigureIndexManagement = func(ctx spi.ComponentContext, namespace string) error {
	cr := ctx.EffectiveCR()
	log := ctx.Log()
	if !vzconfig.IsElasticsearchEnabled(cr) {
		log.Debug("Skipping DataStream setup, backend is disabled")
		return nil
	}
	pods, ok := isOpenSearchReady(ctx, namespace)
	if !ok {
		return fmt.Errorf("cannot create data stream, %s container is not ready yet", containerName)
	}
	pod := &pods[0]
	return doSetupViaOpenSearchAPI(ctx, pod)
}

//doSetupViaOpenSearchAPI creates the ISM Policy and Index Template
func doSetupViaOpenSearchAPI(ctx spi.ComponentContext, pod *corev1.Pod) error {
	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return err
	}

	var policies = vzapi.RetentionPolicies{}
	cr := ctx.EffectiveCR()
	if cr.Spec.Components.Elasticsearch != nil {
		policies = cr.Spec.Components.Elasticsearch.RetentionPolicies
	}

	// Create Retention Policy for Verrazzano Applications
	if err := putRetententionPolicy(cfg, cli, pod, policies.Application, "verrazzano-application", applicationISMPayloadTemplate); err != nil {
		return err
	}
	// Create ISM Policy for Verrazzano System
	if err := putRetententionPolicy(cfg, cli, pod, policies.System, "verrazzano-system", systemISMPayloadTemplate); err != nil {
		return err
	}

	putTemplatePayload := fmt.Sprintf("_index_template/verrazzano-data-stream -d '%s'", indexTemplatePayload)
	return doPUT(cfg, cli, pod, putTemplatePayload)
}

func putRetententionPolicy(cfg *rest.Config, cli kubernetes.Interface, pod *corev1.Pod, retentionPolicy vzapi.RetentionPolicy, policyName, template string) error {
	// Skip ISM Creation if disabled
	if retentionPolicy.Enabled != nil && !*retentionPolicy.Enabled {
		return nil
	}
	// Check if Policy exists or not
	getCommand := makeBashCommand(fmt.Sprintf("curl 'localhost:9200/_plugins/_ism/policies/%s'", policyName))
	getResponse, _, err := k8sutil.ExecPod(cli, cfg, pod, containerName, getCommand)
	if err != nil {
		return err
	}
	serverPolicy := &ISMPolicy{}
	if err := json.Unmarshal([]byte(getResponse), serverPolicy); err != nil {
		return err
	}

	// Create payload for updating ISM Policy
	payload, err := formatISMPayload(retentionPolicy, template)
	if err != nil {
		return err
	}

	// If Policy doesn't exist, PUT it. If Policy exists, POST it.
	var cmd string
	if serverPolicy.Status == notFound {
		cmd = fmt.Sprintf("curl -X PUT -H 'Content-Type: application/json' 'localhost:9200/_plugins/_ism/policies/%s' -d '%s'", policyName, payload)
	} else {
		cmd = fmt.Sprintf("curl -X POST -H 'Content-Type: application/json' 'localhost:9200/_plugins/_ism/policies/%s?if_seq_no=%d&if_primary_term=%d' -d '%s'",
			policyName,
			serverPolicy.SequenceNumber,
			serverPolicy.PrimaryTerm,
			payload,
		)
	}
	containerCommand := makeBashCommand(cmd)
	_, _, err = k8sutil.ExecPod(cli, cfg, pod, containerName, containerCommand)
	return err
}

func formatISMPayload(retentionPolicy vzapi.RetentionPolicy, payload string) (string, error) {
	tmpl, err := template.New("lifecycleManagement").
		Option("missingkey=error").
		Parse(payload)
	if err != nil {
		return "", err
	}
	values := make(map[string]string)
	putOrDefault := func(value *string, key, defaultValue string) {
		if value == nil {
			values[key] = defaultValue
		} else {
			values[key] = *value
		}
	}
	putOrDefault(retentionPolicy.MinAge, minIndexAge, defaultMinIndexAge)
	buffer := &bytes.Buffer{}
	if err := tmpl.Execute(buffer, values); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// fixupElasticSearchReplicaCount fixes the replica count set for single node Elasticsearch cluster
func fixupElasticSearchReplicaCount(ctx spi.ComponentContext, namespace string) error {
	// Only apply this fix to clusters with Elasticsearch enabled.
	if !vzconfig.IsElasticsearchEnabled(ctx.EffectiveCR()) {
		ctx.Log().Debug("Elasticsearch Post Upgrade: Replica count update unnecessary on managed cluster.")
		return nil
	}

	// Only apply this fix to clusters being upgraded from a source version before 1.1.0.
	ver1_1_0, err := semver.NewSemVersion("v1.1.0")
	if err != nil {
		return err
	}
	sourceVer, err := semver.NewSemVersion(ctx.ActualCR().Status.Version)
	if err != nil {
		ctx.Log().Errorf("Elasticsearch Post Upgrade: Invalid source Verrazzano version: %s", err)
		return err
	}
	if sourceVer.IsGreatherThan(ver1_1_0) || sourceVer.IsEqualTo(ver1_1_0) {
		ctx.Log().Debug("Elasticsearch Post Upgrade: Replica count update unnecessary for source Verrazzano version %v.", sourceVer.ToString())
		return nil
	}

	// Wait for an Elasticsearch (i.e., label app=system-es-master) pod with container (i.e. es-master) to be ready.
	pods, err := waitForReadyESContainers(ctx, namespace)
	if err != nil {
		return err
	}
	pod := pods[0]

	// Find the Elasticsearch HTTP control container port.
	httpPort, err := getNamedContainerPortOfContainer(pod, containerName, portName)
	if err != nil {
		ctx.Log().Errorf("Failed to find HTTP port of Elasticsearch container during post-upgrade: %v", err)
		return err
	}
	if httpPort <= 0 {
		err := fmt.Errorf("no port found")
		ctx.Log().Errorf("Failed to find Elasticsearch port during post-upgrade: %v", err)
		return err
	}

	// Set the the number of replicas for the Verrazzano indices
	// to something valid in single node Elasticsearch cluster
	ctx.Log().Debug("Elasticsearch Post Upgrade: Getting the health of the Elasticsearch cluster")
	getCmd := execCommand("kubectl", "exec", pod.Name, "-n", namespace, "-c", containerName, "--", "sh", "-c",
		fmt.Sprintf("curl -v -XGET -s -k --fail http://localhost:%d/_cluster/health", httpPort))
	output, err := getCmd.Output()
	if err != nil {
		ctx.Log().Errorf("Elasticsearch Post Upgrade: Error getting the Elasticsearch cluster health: %s", err)
		return err
	}
	ctx.Log().Debugf("Elasticsearch Post Upgrade: Output of the health of the Elasticsearch cluster %s", string(output))
	// If the data node count is seen as 1 then the node is considered as single node cluster
	if strings.Contains(string(output), `"number_of_data_nodes":1,`) {
		// Login to Elasticsearch and update index settings for single data node elasticsearch cluster
		putCmd := execCommand("kubectl", "exec", pod.Name, "-n", namespace, "-c", containerName, "--", "sh", "-c",
			fmt.Sprintf(`curl -v -XPUT -d '{"index":{"auto_expand_replicas":"0-1"}}' --header 'Content-Type: application/json' -s -k --fail http://localhost:%d/%s/_settings`, httpPort, indexPattern))
		_, err = putCmd.Output()
		if err != nil {
			ctx.Log().Errorf("Elasticsearch Post Upgrade: Error logging into Elasticsearch: %s", err)
			return err
		}
		ctx.Log().Debug("Elasticsearch Post Upgrade: Successfully updated Elasticsearch index settings")
	}
	ctx.Log().Debug("Elasticsearch Post Upgrade: Completed successfully")
	return nil
}

func waitForReadyESContainers(ctx spi.ComponentContext, namespace string) ([]corev1.Pod, error) {
	// Wait for an Elasticsearch (i.e., label app=system-es-master) pod with container (i.e. es-master) to be ready.
	pods, err := waitForPodsWithReadyContainer(ctx.Client(), 15*time.Second, 5*time.Minute, containerName, clipkg.MatchingLabels{"app": workloadName}, clipkg.InNamespace(namespace))
	if err != nil {
		ctx.Log().Errorf("Elasticsearch Post Upgrade: Error getting the Elasticsearch pods: %s", err)
		return nil, err
	}
	if len(pods) == 0 {
		err := fmt.Errorf("no pods found")
		ctx.Log().Errorf("Elasticsearch Post Upgrade: Failed to find Elasticsearch pods: %s", err)
		return nil, err
	}

	return pods, nil
}

func getNamedContainerPortOfContainer(pod corev1.Pod, containerName string, portName string) (int32, error) {
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			for _, port := range container.Ports {
				if port.Name == portName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, fmt.Errorf("no port named %s found in container %s of pod %s", portName, containerName, pod.Name)
}

func getPodsWithReadyContainer(client clipkg.Client, containerName string, podSelectors ...clipkg.ListOption) ([]corev1.Pod, error) {
	pods := []corev1.Pod{}
	list := &corev1.PodList{}
	err := client.List(context.TODO(), list, podSelectors...)
	if err != nil {
		return pods, err
	}
	for _, pod := range list.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name == containerName && containerStatus.Ready {
				pods = append(pods, pod)
			}
		}
	}
	return pods, err
}

func waitForPodsWithReadyContainer(client clipkg.Client, retryDelay time.Duration, timeout time.Duration, containerName string, podSelectors ...clipkg.ListOption) ([]corev1.Pod, error) {
	start := time.Now()
	for {
		pods, err := getPodsWithReadyContainer(client, containerName, podSelectors...)
		if err == nil && len(pods) > 0 {
			return pods, err
		}
		if time.Since(start) >= timeout {
			return pods, err
		}
		time.Sleep(retryDelay)
	}
}
