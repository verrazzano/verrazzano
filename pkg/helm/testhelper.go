// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"io"
)

type CreateReleaseFnType func(name string, releaseStatus release.Status) *release.Release

func CreateActionConfig(includeRelease bool, releaseName string, releaseStatus release.Status, log vzlog.VerrazzanoLogger, createReleaseFn CreateReleaseFnType) (*action.Configuration, error) {

	registryClient, err := registry.NewClient()
	if err != nil {
		return nil, err
	}

	cfg := &action.Configuration{
		Releases:       storage.Init(driver.NewMemory()),
		KubeClient:     &fake.FailingKubeClient{PrintingKubeClient: fake.PrintingKubeClient{Out: io.Discard}},
		Capabilities:   chartutil.DefaultCapabilities,
		RegistryClient: registryClient,
		Log:            log.Infof,
	}
	if includeRelease {
		testRelease := createReleaseFn(releaseName, releaseStatus)
		err = cfg.Releases.Create(testRelease)
		if err != nil {
			return nil, err
		}
	}
	return cfg, nil
}
