// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package common

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1beta2 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta2"
)

func FindPlatformModuleVersion(log vzlog.VerrazzanoLogger, module installv1beta2.Module, pd *installv1beta2.PlatformDefinition) (string, bool) {
	moduleInfo, ok := FindModuleInfo(module.Name, pd)
	if ok {
		return moduleInfo.DefaultVersion, true
	}
	return "", false
}

func FindModuleInfo(modName string, pd *installv1beta2.PlatformDefinition) (installv1beta2.ChartVersion, bool) {
	for _, modInfo := range pd.Spec.CRDVersions {
		if modInfo.Name == modName {
			return modInfo, true
		}
	}
	for _, modInfo := range pd.Spec.OperatorVersions {
		if modInfo.Name == modName {
			return modInfo, true
		}
	}
	for _, modInfo := range pd.Spec.ModuleVersions {
		if modInfo.Name == modName {
			return modInfo, true
		}
	}
	return installv1beta2.ChartVersion{}, false
}
