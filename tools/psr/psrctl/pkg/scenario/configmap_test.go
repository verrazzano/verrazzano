// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

func Test(t *testing.T) {
	sc := Scenario{
		HelmReleases: []types.NamespacedName{{
			Namespace: "ns1",
			Name:      "name1",
		}},
		ScenarioManifest: &ScenarioManifest{
			Name:        "os1-name",
			ID:          "os1-id",
			Description: "test scenario",
			Usecases: []Usecase{{
				UsecasePath:  "ucPath",
				OverrideFile: "ucOverride",
				Description:  "desc",
			}},
			ScenarioUsecaseOverridesDir: "overridedir",
		},
	}

	saveScenario(vzlog.DefaultLogger(), sc, "ns1")
}
