// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	PsrPrefix = "psr"
)

type yamlMap struct {
	yMap map[string]string
}

func saveScenario(log vzlog.VerrazzanoLogger, scenario Scenario, namespace string) (*corev1.ConfigMap, error) {
	client, err := k8sutil.GetCoreV1Client(log)
	if err != nil {
		return nil, log.ErrorfNewErr("Failed to get CoreV1 client: %v", err)
	}

	y, err := yaml.Marshal(scenario)
	if err != nil {
		return nil, log.ErrorfNewErr("Failed to mashal scenario to YAML: %v", err)
	}
	// convert to base64
	encoded := base64.StdEncoding.EncodeToString(y)

	cmIn := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildConfigmapName(scenario.ID),
			Namespace: namespace,
			Labels: map[string]string{
				LabelScenarioKey:   "true",
				LabelScenarioIdKey: scenario.ScenarioManifest.ID},
		},
		Data: map[string]string{"scenario": encoded},
	}

	cmNew, err := client.ConfigMaps(namespace).Create(context.TODO(), &cmIn, metav1.CreateOptions{})
	if err != nil {
		return nil, log.ErrorfNewErr("Failed to create scenario ConfigMap %s/%s: %v", cmIn.Namespace, cmIn.Name, err)
	}
	return cmNew, nil
}

func buildConfigmapName(name string) string {
	return fmt.Sprintf("%s%s%s", PsrPrefix, "-", name)
}
