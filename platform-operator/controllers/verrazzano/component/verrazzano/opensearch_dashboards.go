// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"reflect"
	"regexp"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"text/template"
)

const (
	OSDashboardContainerName = "kibana"
	OSDashboardSystemName    = "system-kibana"
	OSDashboardServicePort   = "5601"
)
const updatePatternPayload = `{
   "attributes": {
     "title": "{{ .IndexPattern }}"
   }
 }
`

type PatternInput struct {
	IndexPattern string
}

func getOpenSearchDashboardPod(ctx spi.ComponentContext, namespace string) (*corev1.Pod, error) {
	cr := ctx.EffectiveCR()
	if !vzconfig.IsElasticsearchEnabled(cr) {
		ctx.Log().Debug("OpenSearchDashboards is disabled")
		return nil, nil
	}
	pods, ok := isOpenSearchDashboardReady(ctx, namespace)
	if !ok {
		return nil, fmt.Errorf("%s container is not ready yet", OSDashboardContainerName)
	}
	return &pods[0], nil
}

// isOpenSearchDashboardReady checks if the OpenSearchDashboard is ready
func isOpenSearchDashboardReady(ctx spi.ComponentContext, namespace string) ([]corev1.Pod, bool) {
	pods, err := getPodsWithReadyContainer(ctx.Client(), OSDashboardContainerName,
		clipkg.MatchingLabels{"app": OSDashboardSystemName}, clipkg.InNamespace(namespace))
	if err != nil {
		return nil, false
	}
	if len(pods) < 1 {
		return nil, false
	}
	return pods, true
}

func getPatterns(log vzlog.VerrazzanoLogger, cfg *rest.Config, cli kubernetes.Interface, pod *corev1.Pod) (map[string]string, error) {
	if pod == nil {
		log.Infof("OpenSearch Dashboards pod is not configured to run. Skipping the post upgrade step for OpenSearch Dashboards")
		return nil, nil
	}
	getIndexPatterns := makeBashCommand(fmt.Sprintf("curl -X GET -k --fail 'http://localhost:%s/api/saved_objects/_find?type=index-pattern&fields=title'", OSDashboardServicePort))
	getResponse, _, err := k8sutil.ExecPod(cli, cfg, pod, OSDashboardContainerName, getIndexPatterns)
	if err != nil {
		return nil, log.ErrorfNewErr("OpenSearch Dashboards post upgrade: Error in getting index patterns: %v", err)
	}
	if getResponse == "" {
		log.Debugf("OpenSearch Dashboards post upgrade: Empty response for get index patterns")
		return nil, nil
	}
	log.Debugf("OpenSearch Dashboards post upgrade: Get index patterns response %v", getResponse)
	var responseMap map[string]interface{}
	if err := json.Unmarshal([]byte(getResponse), &responseMap); err != nil {
		log.Errorf("OpenSearch Dashboards post upgrade: Error unmarshalling index patterns response body: %v", err)
	}
	patterns := make(map[string]string)
	if responseMap["saved_objects"] != nil {
		savedObjects := reflect.ValueOf(responseMap["saved_objects"])
		for i := 0; i < savedObjects.Len(); i++ {
			log.Debugf("OpenSearch Dashboards post upgrade: Index pattern details: %v", savedObjects.Index(i))
			savedObject := savedObjects.Index(i).Interface().(map[string]interface{})
			attributes := savedObject["attributes"].(map[string]interface{})
			title := attributes["title"].(string)
			id := savedObject["id"]
			patterns[id.(string)] = title
		}
	}
	log.Debugf("OpenSearch Dashboards post upgrade: Found index patterns in OpenSearch Dashboards %v", patterns)
	return patterns, nil
}

func updatePatterns(ctx spi.ComponentContext, cfg *rest.Config, cli kubernetes.Interface, namespace string) error {
	OSPod, err := getOpenSearchDashboardPod(ctx, namespace)
	if err != nil {
		return err
	}
	dashboardPatterns, err := getPatterns(ctx.Log(), cfg, cli, OSPod)
	if err != nil {
		return err
	}
	for id, originalPattern := range dashboardPatterns {
		updatedPattern := constructUpdatedPattern(originalPattern)
		if id == "" || (originalPattern == updatedPattern) {
			continue
		}
		// Invoke update index pattern API
		err = executeUpdate(ctx.Log(), cfg, cli, OSPod, id, originalPattern, updatedPattern)
		if err != nil {
			ctx.Log().Infof("OpenSearch Dashboards post upgrade: Updating index pattern failed: %v", err)
			return err
		}
	}
	return nil
}

func executeUpdate(log vzlog.VerrazzanoLogger, cfg *rest.Config, cli kubernetes.Interface, pod *corev1.Pod,
	id string, originalPattern string, updatedPattern string) error {
	input := PatternInput{IndexPattern: updatedPattern}
	payload, err := formatPatternPayload(input, updatePatternPayload)
	if err != nil {
		return err
	}

	log.Infof("OpenSearch Dashboards post upgrade: Replacing index pattern %s with %s in OpenSearch Dashboards", originalPattern, updatedPattern)
	cmd := fmt.Sprintf("curl -k --fail -X PUT -H 'Content-Type: application/json' -H 'osd-xsrf: true' 'localhost:5601/api/saved_objects/index-pattern/%s' -d '%s'", id, payload)
	log.Debugf("OpenSearch Dashboards post upgrade: Executing update saved object API %s", cmd)
	containerCommand := makeBashCommand(cmd)
	response, _, err := k8sutil.ExecPod(cli, cfg, pod, OSDashboardContainerName, containerCommand)
	log.Debugf("OpenSearch Dashboards post upgrade: Update index pattern API response %v", response)
	if err != nil {
		return err
	}
	return nil
}

func formatPatternPayload(input PatternInput, payload string) (string, error) {
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

func constructUpdatedPattern(originalPattern string) string {
	var updatedPattern []string
	patternList := strings.Split(originalPattern, ",")
	for _, eachPattern := range patternList {
		if strings.HasPrefix(eachPattern, "verrazzano-") && eachPattern != "verrazzano-*" {
			// To match the exact pattern, add ^ in the beginning and $ in the end
			regexpString := convertToRegexp(eachPattern)
			systemIndexMatch := isSystemIndexMatch(regexpString)
			if systemIndexMatch {
				updatedPattern = append(updatedPattern, systemDataStreamName)
			}
			isNamespaceIndexMatch, _ := regexp.MatchString(regexpString, "verrazzano-namespace-")
			if isNamespaceIndexMatch {
				updatedPattern = append(updatedPattern, "verrazzano-application-*")
			} else if strings.HasPrefix(eachPattern, "verrazzano-namespace-") {
				// If the pattern matches system index and no * present in the pattern, then it is considered as only
				// system index
				if systemIndexMatch && !strings.Contains(eachPattern, "*") {
					continue
				}
				updatedPattern = append(updatedPattern, strings.Replace(eachPattern, "verrazzano-namespace-", "verrazzano-application-", 1))
			}
		} else {
			updatedPattern = append(updatedPattern, eachPattern)
		}
	}
	return strings.Join(updatedPattern, ",")
}

func isSystemIndexMatch(pattern string) bool {
	logStashIndex, _ := regexp.MatchString(pattern, "verrazzano-logstash-")
	systemJournalIndex, _ := regexp.MatchString(pattern, "verrazzano-systemd-journal")
	if logStashIndex || systemJournalIndex {
		return true
	}
	for _, namespace := range systemNamespaces {
		systemIndex, _ := regexp.MatchString(pattern, "verrazzano-namespace-"+namespace)
		if systemIndex {
			return true
		}
	}
	return false
}

// convertToRegexp converts index pattern to a regular expression pattern.
func convertToRegexp(pattern string) string {
	var result strings.Builder
	// Add ^ at the beginning
	result.WriteString("^")
	for i, literal := range strings.Split(pattern, "*") {

		// Replace * with .*
		if i > 0 {
			result.WriteString(".*")
		}

		// Quote any regular expression meta characters in the
		// literal text.
		result.WriteString(regexp.QuoteMeta(literal))
	}
	// Add $ at the end
	result.WriteString("$")
	return result.String()
}
