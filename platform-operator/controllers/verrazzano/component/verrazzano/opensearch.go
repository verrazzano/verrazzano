// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"regexp"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const (
	minIndexAge            = "min_index_age"
	defaultMinIndexAge     = "7d"
	systemDataStreamName   = "verrazzano-system"
	dataStreamTemplateName = "verrazzano-data-stream"
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

const reindexPayload = `{
  "source": {
    "index": "{{ .SourceName }}",
    "query": {
      "range": {
        "@timestamp": {
          "gte": "now-{{ .NumberOfSeconds }}/s",
          "lt": "now/s"
        }
      }
    }
  },
  "dest": {
    "index": "{{ .DestinationName }}",
    "op_type": "create"
  }
}`

const reindexPayloadWithoutQuery = `{
  "source": {
    "index": "{{ .SourceName }}"
  },
  "dest": {
    "index": "{{ .DestinationName }}",
    "op_type": "create"
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

type ReindexInput struct {
	SourceName      string
	DestinationName string
	NumberOfSeconds string
}

type ReindexInputWithoutQuery struct {
	SourceName      string
	DestinationName string
}

// The system namespaces used in Verrazzano
var (
	systemNamespaces = []string{"kube-system", "verrazzano-system", "istio-system", "keycloak", "metallb-system",
		"default", "cert-manager", "local-path-storage", "rancher-operator-system", "fleet-system", "ingress-nginx",
		"cattle-system", "verrazzano-install", "monitoring"}
	agePattern       = "^(?P<number>\\d+)(?P<unit>[yMwdhHms])$"
	reTimeUnit       = regexp.MustCompile(agePattern)
	secondsPerMinute = uint64(60)
	secondsPerHour   = secondsPerMinute * 60
	secondsPerDay    = secondsPerHour * 24
	secondsPerWeek   = secondsPerDay * 7
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
	return doSetupViaOpenSearchAPI(ctx, pod, namespace)
}

//doSetupViaOpenSearchAPI creates the ISM Policy and Index Template
func doSetupViaOpenSearchAPI(ctx spi.ComponentContext, pod *corev1.Pod, namespace string) error {
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
	if err := putRetententionPolicy(cfg, cli, pod, policies.System, "verrazzano-system", "verrazzano-system", systemISMPayloadTemplate); err != nil {
		return err
	}

	// During upgrade, reindex and delete old indices
	if ctx.GetOperation() == constants.UpgradeOperation {
		if err := pruneOldIndices(ctx, cfg, cli, pod, policies, namespace); err != nil {
			return err
		}
	}
	return nil
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

// Reindex old style indices to data streams and delete it
func pruneOldIndices(ctx spi.ComponentContext, cfg *rest.Config, cli kubernetes.Interface, pod *corev1.Pod,
	policies vzapi.RetentionPolicies, namespace string) error {
	ctx.Log().Info("OpenSearch Post Upgrade: Migrating Verrazzano old indices if any to data streams")
	// Make sure that the data stream template is created before re-indexing
	err := verifyDataStreamTemplateExists(ctx.Log(), cfg, cli, pod, dataStreamTemplateName, 2*time.Minute,
		15*time.Second)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed in OpenSearch post upgrade: error in verifying the existence of"+
			" data stream template %s: %v", dataStreamTemplateName, err)
	}
	// Get system indices
	systemIndices, err := getSystemIndices(ctx.Log(), cfg, cli, pod)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed in OpenSearch post upgrade: error in getting the Verrazzano system indices: %v", err)
	}
	// Calculate the number of seconds of past system data that has to be re-indexed
	var noOfSecsOfSystemData string
	if policies.System.Enabled != nil && *policies.System.Enabled {
		noOfSecs, err := calculateSeconds(*policies.System.MinAge)
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed in OpenSearch post upgrade: error in calculating the number of"+
				" seconds of past system logs that has to be re-indexed: %v", err)
		}
		noOfSecsOfSystemData = fmt.Sprintf("%ds", noOfSecs)
	}
	// Reindex and delete old system indices
	err = reindexAndDeleteIndices(ctx.Log(), cfg, cli, pod, systemIndices, true, noOfSecsOfSystemData)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed in OpenSearch post upgrade: error in migrating the old Verrazzano"+
			" system indices to data streams: %v", err)
	}
	// Get application indices
	appIndices, err := getApplicationIndices(ctx.Log(), cfg, cli, pod)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed in OpenSearch post upgrade: error in getting the Verrazzano application indices: %v", err)
	}
	// Calculate the number of seconds of past application data that has to be re-indexed
	var noOfSecsOfAppData string
	if policies.Application.Enabled != nil && *policies.Application.Enabled {
		noOfSecs, err := calculateSeconds(*policies.Application.MinAge)
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed in OpenSearch post upgrade: error in calculating the number of"+
				" seconds of past application logs that has to be re-indexed: %v", err)
		}
		noOfSecsOfAppData = fmt.Sprintf("%ds", noOfSecs)
	}
	// Reindex and delete old application indices
	err = reindexAndDeleteIndices(ctx.Log(), cfg, cli, pod, appIndices, false, noOfSecsOfAppData)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed in OpenSearch post upgrade: error in migrating the old Verrazzano"+
			" application indices to data streams: %v", err)
	}

	// Update index patterns in OpenSearch dashboards
	err = updatePatterns(ctx, cfg, cli, namespace)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed in OpenSearch post upgrade: error in updating index patterns"+
			" in OpenSearch Dashboards: %v", err)
	}
	ctx.Log().Info("OpenSearch Post Upgrade: Migration of Verrazzano old indices to data streams completed successfully")
	return nil
}

func getSystemIndices(log vzlog.VerrazzanoLogger, cfg *rest.Config, cli kubernetes.Interface,
	pod *corev1.Pod) ([]string, error) {
	var indices []string
	indices, err := getIndices(log, cfg, cli, pod)
	if err != nil {
		return nil, err
	}
	var systemIndices []string
	for _, index := range indices {
		if strings.HasPrefix(index, "verrazzano-namespace-") {
			for _, systemNamespace := range systemNamespaces {
				if index == "verrazzano-namespace-"+systemNamespace {
					systemIndices = append(systemIndices, index)
				}
			}
		}
		if strings.Contains(index, "verrazzano-systemd-journal") {
			systemIndices = append(systemIndices, index)
		}
		if strings.HasPrefix(index, "verrazzano-logstash-") {
			systemIndices = append(systemIndices, index)
		}
	}
	log.Debugf("OpenSearch Post Upgrade: Found Verrazzano system indices %v", systemIndices)
	return systemIndices, nil
}

func getApplicationIndices(log vzlog.VerrazzanoLogger, cfg *rest.Config, cli kubernetes.Interface, pod *corev1.Pod) ([]string, error) {
	var indices []string
	indices, err := getIndices(log, cfg, cli, pod)
	if err != nil {
		return nil, err
	}
	var appIndices []string
	for _, index := range indices {
		systemIndex := false
		if strings.HasPrefix(index, "verrazzano-namespace-") {
			for _, systemNamespace := range systemNamespaces {
				if index == "verrazzano-namespace-"+systemNamespace {
					systemIndex = true
					break
				}
			}
			if !systemIndex {
				appIndices = append(appIndices, index)
			}
		}
	}
	log.Debugf("OpenSearch Post Upgrade: Found Verrazzano application indices %v", appIndices)
	return appIndices, nil
}

func getIndices(log vzlog.VerrazzanoLogger, cfg *rest.Config, cli kubernetes.Interface, pod *corev1.Pod) ([]string, error) {
	getVZIndices := makeBashCommand(fmt.Sprintf("curl -XGET -s -k --fail 'http://localhost:%s/_cat/indices/verrazzano-*?format=json'", searchPort))
	getResponse, _, err := k8sutil.ExecPod(cli, cfg, pod, containerName, getVZIndices)
	if err != nil {
		log.Errorf("OpenSearch Post Upgrade: Error getting cluster indices: %v", err)
		return nil, err
	}
	log.Debugf("OpenSearch Post Upgrade: Response body %v", getResponse)
	var indices []map[string]interface{}
	if err := json.Unmarshal([]byte(getResponse), &indices); err != nil {
		log.Errorf("OpenSearch Post Upgrade: Error unmarshalling indices response body: %v", err)
		return nil, err
	}
	var indexNames []string
	for _, index := range indices {
		val, found := index["index"]
		if !found {
			log.Errorf("OpenSearch Post Upgrade: Not able to find the name of the index: %v", index)
			return nil, err
		}
		indexNames = append(indexNames, val.(string))

	}
	log.Debugf("OpenSearch Post Upgrade: Found Verrazzano indices %v", indices)
	return indexNames, nil
}

func reindexAndDeleteIndices(log vzlog.VerrazzanoLogger, cfg *rest.Config, cli kubernetes.Interface, pod *corev1.Pod,
	indices []string, isSystemIndex bool, retentionDays string) error {
	for _, index := range indices {
		var dataStreamName string
		if isSystemIndex {
			dataStreamName = systemDataStreamName
		} else {
			dataStreamName = strings.Replace(index, "verrazzano-namespace", "verrazzano-application", 1)
		}
		log.Infof("OpenSearch Post Upgrade: Reindexing logs from index %v to data stream %s", index, dataStreamName)
		err := reindexToDataStream(log, cfg, cli, pod, index, dataStreamName, retentionDays)
		if err != nil {
			return err
		}
		log.Infof("OpenSearch Post Upgrade: Deleting old index %v", index)
		err = deleteIndex(log, cfg, cli, pod, index)
		if err != nil {
			return err
		}
		log.Infof("OpenSearch Post Upgrade: Deleted old index %v successfully", index)
	}
	return nil
}

func reindexToDataStream(log vzlog.VerrazzanoLogger, cfg *rest.Config, cli kubernetes.Interface, pod *corev1.Pod,
	sourceName string, destName string, retentionDays string) error {
	var payload string
	var err error
	if retentionDays == "" {
		input := ReindexInputWithoutQuery{SourceName: sourceName, DestinationName: destName}
		payload, err = formatReindexPayloadWithoutQuery(input, reindexPayloadWithoutQuery)
	} else {
		input := ReindexInput{SourceName: sourceName, DestinationName: destName, NumberOfSeconds: retentionDays}
		payload, err = formatReindexPayload(input, reindexPayload)
	}
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("curl -k --fail -X POST -H 'Content-Type: application/json' 'localhost:9200/_reindex' -d '%s'", payload)
	log.Debugf("OpenSearch Post Upgrade: Executing reindex API %s", cmd)
	containerCommand := makeBashCommand(cmd)
	stdOut, stdErr, err := k8sutil.ExecPod(cli, cfg, pod, containerName, containerCommand)
	if err != nil {
		log.Errorf("OpenSearch Post Upgrade: Reindex from %s to %s failed: stdout = %s: stderr = %s", sourceName, destName, stdOut, stdErr)
		return err
	}
	log.Debugf("OpenSearch Post Upgrade: Reindex from %s to %s API response %s", sourceName, destName, stdOut)
	return nil
}

func deleteIndex(log vzlog.VerrazzanoLogger, cfg *rest.Config, cli kubernetes.Interface, pod *corev1.Pod,
	indexName string) error {
	cmd := fmt.Sprintf("curl -k --fail -X DELETE -H 'Content-Type: application/json' 'localhost:9200/%s'", indexName)
	log.Debugf("OpenSearch Post Upgrade: Executing delete API %s", cmd)
	containerCommand := makeBashCommand(cmd)
	response, _, err := k8sutil.ExecPod(cli, cfg, pod, containerName, containerCommand)
	log.Debugf("OpenSearch Post Upgrade: Delete API response %s", response)
	return err
}

func verifyDataStreamTemplateExists(log vzlog.VerrazzanoLogger, cfg *rest.Config, cli kubernetes.Interface,
	pod *corev1.Pod, templateName string, retryDelay time.Duration, timeout time.Duration) error {
	cmd := fmt.Sprintf("curl -k --fail -X GET 'localhost:9200/_index_template/%s'", templateName)
	log.Debugf("OpenSearch Post Upgrade: Executing get template API %s", cmd)
	containerCommand := makeBashCommand(cmd)
	start := time.Now()
	for {
		response, _, err := k8sutil.ExecPod(cli, cfg, pod, containerName, containerCommand)
		if err != nil {
			return err
		}
		if strings.Contains(response, `"name":"`+templateName) {
			return nil
		}
		if time.Since(start) >= timeout {
			return log.ErrorfNewErr("OpenSearch post upgrade: Time out in verifying the existence of "+
				"data stream template %s", templateName)
		}
		time.Sleep(retryDelay)
	}
}

func formatReindexPayload(input ReindexInput, payload string) (string, error) {
	tmpl, err := template.New("reindex").
		Option("missingkey=error").
		Parse(payload)
	if err != nil {
		return "", err
	}
	buffer := &bytes.Buffer{}
	if err := tmpl.Execute(buffer, input); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func formatReindexPayloadWithoutQuery(input ReindexInputWithoutQuery, payload string) (string, error) {
	tmpl, err := template.New("reindexWithoutQuery").
		Option("missingkey=error").
		Parse(payload)
	if err != nil {
		return "", err
	}
	buffer := &bytes.Buffer{}
	if err := tmpl.Execute(buffer, input); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func calculateSeconds(age string) (uint64, error) {
	match := reTimeUnit.FindStringSubmatch(age)
	if match == nil || len(match) < 2 {
		return 0, fmt.Errorf("unable to convert %s to seconds due to invalid format", age)
	}
	n := match[1]
	number, err := strconv.ParseUint(n, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("unable to parse the specified time unit %s", n)
	}
	switch match[2] {
	case "w":
		return number * secondsPerWeek, nil
	case "d":
		return number * secondsPerDay, nil
	case "h", "H":
		return number * secondsPerHour, nil
	case "m":
		return number * secondsPerMinute, nil
	case "s":
		return number, nil
	}
	return 0, fmt.Errorf("conversion to seconds for time unit %s is unsupported", match[2])
}
