// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	yaml2 "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func saveScenario(log vzlog.VerrazzanoLogger, scenario Scenario, namespace string) error {

	yaml, err := yaml2.Marshal(scenario)
	if err != nil {
		return log.ErrorfNewErr("Failed to mashal scenario to YAML: %v", err)
	}
	scenario 

	cm  := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            sm.ID,
			Namespace:       namespace,
		},
		Data:       nil,
	}
},
