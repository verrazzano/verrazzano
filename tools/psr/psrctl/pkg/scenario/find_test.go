// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/embedded"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

func Test(t *testing.T) {
	m := Manager{
		Namespace: "default",
		Log:       vzlog.DefaultLogger(),
		Manifest: embedded.PsrManifests{
			ScenarioAbsDir: "./testdata",
		},
	}
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

	m.createScenarioConfigMap(sc)
}
