// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "verrazzano"

const vzDefaultNamespace = constants.VerrazzanoSystemNamespace

var execCommand = exec.Command

// ResolveVerrazzanoNamespace will return the default Verrazzano system namespace unless the namespace
// is specified
func ResolveVerrazzanoNamespace(ns string) string {
	if len(ns) > 0 && ns != "default" {
		return ns
	}
	return vzDefaultNamespace
}

// VerrazzanoPreUpgrade contains code that is run prior to helm upgrade for the Verrazzano helm chart
func VerrazzanoPreUpgrade(log *zap.SugaredLogger, client client.Client, _ string, namespace string, _ string) error {
	return fixupFluentdDaemonset(log, client, namespace)
}

// This function is used to fixup the fluentd daemonset on a managed cluster so that helm upgrade of Verrazzano does
// not fail.  Prior to Verrazzano v1.0.1, the mcagent would change the environment variables CLUSTER_NAME and
// ELASTICSEARCH_URL on a managed cluster to use valueFrom (from a secret) instead of using a Value. The helm chart
// template for the fluentd daemonset expects a Value.
func fixupFluentdDaemonset(log *zap.SugaredLogger, client client.Client, namespace string) error {
	// Get the fluentd daemonset resource
	fluentdNamespacedName := types.NamespacedName{Name: "fluentd", Namespace: namespace}
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
	log.Infof("Updating fluentd daemonset to use valueFrom instead of Value for CLUSTER_NAME and ELASTICSEARCH_URL environment variables")
	err = client.Update(context.TODO(), &daemonSet)

	return err
}

// AppendOverrides appends the image overrides for the monitoring-init-images subcomponent
func AppendOverrides(_ spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}

	imageOverrides, err := bomFile.BuildImageOverrides("monitoring-init-images")
	if err != nil {
		return nil, err
	}

	kvs = append(kvs, imageOverrides...)
	return kvs, nil
}

// fixupElasticSearchReplicaCount fixes the wrong replica count set for single node Elasticsearch cluster
func fixupElasticSearchReplicaCount(ctx spi.ComponentContext, namespace string) error {
	ctx.Log().Debug("Elasticsearch Post Upgrade: Version is %v", ctx.EffectiveCR().Spec.Version)
	// While upgrading to 1.1.0 version, fix the wrong number of replicas seen in single node Elasticsearch cluster
	if ctx.EffectiveCR().Spec.Profile != vzapi.ManagedCluster && strings.HasPrefix(ctx.EffectiveCR().Spec.Version, "v1.1.0") {
		ctx.Log().Info("Elasticsearch Post Upgrade: Getting the health of the Elasticsearch cluster")
		cmd := execCommand("kubectl", "exec", "vmi-system-es-master-0", "-n", namespace, "-c", "es-master", "--", "sh", "-c",
			"curl -v -XGET -s -k  --fail http://localhost:9200/_cluster/health")
		output, err := cmd.Output()
		if err != nil {
			ctx.Log().Errorf("Elasticsearch Post Upgrade: Error getting the Elasticsearch cluster health: %s", err)
			return err
		}
		ctx.Log().Info("Elasticsearch Post Upgrade: Output of the health of the Elasticsearch cluster %v", string(output))
		// If the data node count is seen as 1 then the node is considered as single node cluster
		if strings.Contains(string(output), "\"number_of_data_nodes\":1,") {
			// Login to Elasticsearch and update index settings for single data node elasticsearch cluster
			putCmd := execCommand("kubectl", "exec", "vmi-system-es-master-0", "-n", namespace, "-c", "es-master", "--", "sh", "-c",
				"curl -v -XPUT -d '{\"index\":{\"auto_expand_replicas\":\"0-1\"}}' --header 'Content-Type: application/json' -s -k  --fail http://localhost:9200/verrazzano-*/_settings")
			_, err = putCmd.Output()
			if err != nil {
				ctx.Log().Errorf("Elasticsearch Post Upgrade: Error logging into Elasticsearch: %s", err)
				return err
			}
			ctx.Log().Info("Elasticsearch Post Upgrade: Successfully updated Elasticsearch index settings")
		}
	}
	ctx.Log().Info("Elasticsearch Post Upgrade: Completed Successfully")
	return nil
}
