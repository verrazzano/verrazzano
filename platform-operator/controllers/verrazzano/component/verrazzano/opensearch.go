// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"text/template"
	"time"
)

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
            "verrazzano-application*",
			"verrazzano-logstash*"
          ],
          "priority": 1
        }
    }
}`

type (
	ISMPolicy struct {
		ID             string `json:"_id"`
		PrimaryTerm    int    `json:"_primary_term"`
		SequenceNumber int    `json:"_seq_no"`
		Status         int    `json:"status"`
	}
	uriComponents struct {
		host   string
		port   string
		scheme string
	}
)

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
	if err := putRetententionPolicy(cfg, cli, pod, policies.Application, "verrazzano-application", "verrazzano-application*", applicationISMPayloadTemplate); err != nil {
		return err
	}

	// Create ISM Policy for Verrazzano System
	return putRetententionPolicy(cfg, cli, pod, policies.System, "verrazzano-system", "verrazzano-system", systemISMPayloadTemplate)
}

func putRetententionPolicy(cfg *rest.Config, cli kubernetes.Interface, pod *corev1.Pod, retentionPolicy vzapi.RetentionPolicy, policyName, policyIndexPattern, template string) error {
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
	// Create the ISM Policy
	createResponse, _, err := k8sutil.ExecPod(cli, cfg, pod, containerName, containerCommand)
	if err != nil {
		return err
	}
	createdPolicy := &ISMPolicy{}
	if err := json.Unmarshal([]byte(createResponse), createdPolicy); err != nil {
		return err
	}

	// Apply the ISM policy to any existing indexes. This is required if the index was created before the ISM Policy,
	// which could be due to timing.
	addPolicyCmd := fmt.Sprintf(`curl -X POST -H 'Content-Type: application/json' 'localhost:9200/_plugins/_ism/add/%s' -d '{"policy_id": "%s"}'`,
		policyIndexPattern,
		createdPolicy.ID)
	_, _, err = k8sutil.ExecPod(cli, cfg, pod, containerName, makeBashCommand(addPolicyCmd))
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

// fixupOpenSearchReplicaCount fixes the replica count set for single node OpenSearch cluster
func fixupOpenSearchReplicaCount(ctx spi.ComponentContext, namespace string) error {
	// Only apply this fix to clusters with OpenSearch enabled.
	if !vzconfig.IsElasticsearchEnabled(ctx.EffectiveCR()) {
		ctx.Log().Debug("OpenSearch Post Upgrade: Replica count update unnecessary on managed cluster.")
		return nil
	}

	// Only apply this fix to clusters being upgraded from a source version before 1.1.0.
	ver1_1_0, err := semver.NewSemVersion("v1.1.0")
	if err != nil {
		return err
	}
	sourceVer, err := semver.NewSemVersion(ctx.ActualCR().Status.Version)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed OpenSearch post-upgrade: Invalid source Verrazzano version: %v", err)
	}
	if sourceVer.IsGreatherThan(ver1_1_0) || sourceVer.IsEqualTo(ver1_1_0) {
		ctx.Log().Debug("OpenSearch Post Upgrade: Replica count update unnecessary for source Verrazzano version %v.", sourceVer.ToString())
		return nil
	}

	// Wait for an OpenSearch (i.e., label app=system-es-master) pod with container (i.e. es-master) to be ready.
	pods, err := waitForReadyOSContainers(ctx, namespace)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed getting the OpenSearch pods during post-upgrade: %v", err)
	}
	if len(pods) == 0 {
		return ctx.Log().ErrorfNewErr("Failed to find OpenSearch pods during post-upgrade: %v", err)
	}
	pod := pods[0]

	// Find the OpenSearch HTTP control container port.
	httpPort, err := getNamedContainerPortOfContainer(pod, containerName, portName)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to find HTTP port of OpenSearch container during post-upgrade: %v", err)
	}
	if httpPort <= 0 {
		return ctx.Log().ErrorfNewErr("Failed to find OpenSearch port during post-upgrade: %v", err)
	}

	// Set the the number of replicas for the Verrazzano indices
	// to something valid in single node OpenSearch cluster
	ctx.Log().Debug("OpenSearch Post Upgrade: Getting the health of the OpenSearch cluster")
	getCmd := execCommand("kubectl", "exec", pod.Name, "-n", namespace, "-c", containerName, "--", "sh", "-c",
		fmt.Sprintf("curl -v -XGET -s -k --fail http://localhost:%d/_cluster/health", httpPort))
	output, err := getCmd.Output()
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed in OpenSearch post upgrade: error getting the OpenSearch cluster health: %v", err)
	}
	ctx.Log().Debugf("OpenSearch Post Upgrade: Output of the health of the OpenSearch cluster %s", string(output))
	// If the data node count is seen as 1 then the node is considered as single node cluster
	if strings.Contains(string(output), `"number_of_data_nodes":1,`) {
		// Login to OpenSearch and update index settings for single data node OpenSearch cluster
		putCmd := execCommand("kubectl", "exec", pod.Name, "-n", namespace, "-c", containerName, "--", "sh", "-c",
			fmt.Sprintf(`curl -v -XPUT -d '{"index":{"auto_expand_replicas":"0-1"}}' --header 'Content-Type: application/json' -s -k --fail http://localhost:%d/%s/_settings`, httpPort, indexPattern))
		_, err = putCmd.Output()
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed in OpenSearch post-upgrade: Error logging into OpenSearch: %v", err)
		}
		ctx.Log().Debug("OpenSearch Post Upgrade: Successfully updated OpenSearch index settings")
	}
	ctx.Log().Debug("OpenSearch Post Upgrade: Completed successfully")
	return nil
}

func waitForReadyOSContainers(ctx spi.ComponentContext, namespace string) ([]corev1.Pod, error) {
	// Wait for an OpenSearch (i.e., label app=system-es-master) pod with container (i.e. es-master) to be ready.
	pods, err := waitForPodsWithReadyContainer(ctx.Client(), 15*time.Second, 5*time.Minute, containerName, clipkg.MatchingLabels{"app": workloadName}, clipkg.InNamespace(namespace))
	if err != nil {
		ctx.Log().Errorf("OpenSearch Post Upgrade: Error getting the OpenSearch pods: %s", err)
		return nil, err
	}
	if len(pods) == 0 {
		err := fmt.Errorf("no pods found")
		ctx.Log().Errorf("OpenSearch Post Upgrade: Failed to find OpenSearch pods: %s", err)
		return nil, err
	}

	return pods, nil
}
