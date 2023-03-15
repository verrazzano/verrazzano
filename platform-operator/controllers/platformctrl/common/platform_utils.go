// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package common

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	platformapi "github.com/verrazzano/verrazzano/platform-operator/apis/platform/v1alpha1"
)

func FindPlatformModuleVersion(log vzlog.VerrazzanoLogger, module platformapi.Module, pd *platformapi.PlatformDefinition) (string, error) {
	moduleInfo, ok := FindModuleInfo(module.Name, pd)
	if ok {
		return moduleInfo.DefaultVersion, nil
	}
	return "", log.ErrorfThrottledNewErr("Module info not found in platform definition for %s", module.Name)
}

func FindModuleInfo(modName string, pd *platformapi.PlatformDefinition) (platformapi.ChartVersion, bool) {
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
	return platformapi.ChartVersion{}, false
}
