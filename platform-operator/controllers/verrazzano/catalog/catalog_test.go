// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package catalog

import (
	"fmt"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"io"
	"k8s.io/utils/strings/slices"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzString "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
)

const catalogPath = "../../../manifests/catalog/catalog.yaml"
const bomPath = "../../../verrazzano-bom.json"
const testBOMPath = "../testdata/test_bom.json"

const targetBranch = "master"

var remoteBOMPath = fmt.Sprintf("https://raw.githubusercontent.com/verrazzano/verrazzano/%s/platform-operator/verrazzano-bom.json", targetBranch)
var remoteCatalogPath = fmt.Sprintf("https://raw.githubusercontent.com/verrazzano/verrazzano/%s/platform-operator/manifests/catalog/catalog.yaml", targetBranch)

type bomSubcomponentOverrides struct {
	subcomponentName string
	imageName        string
}

var modulesNotInBom = []string{
	"verrazzano-grafana-dashboards",
	"verrazzano-network-policies",
	"cluster-issuer",
	"fluentbit-opensearch-output",
}

var subcomponentOverrides = map[string][]bomSubcomponentOverrides{
	"fluentbit-opensearch-output": {{subcomponentName: "fluent-operator", imageName: "fluent-bit"}},
	"opensearch":                  {{subcomponentName: "verrazzano-monitoring-operator", imageName: "opensearch"}},
	"opensearch-dashboards":       {{subcomponentName: "verrazzano-monitoring-operator", imageName: "opensearch-dashboards"}},
	"grafana":                     {{subcomponentName: "verrazzano-monitoring-operator", imageName: "grafana"}},
	"cluster-api": {
		{subcomponentName: "capi-cluster-api", imageName: "cluster-api-controller"},
		{subcomponentName: "capi-oci", imageName: "cluster-api-oci-controller"},
		{subcomponentName: "capi-ocne", imageName: "cluster-api-ocne-bootstrap-controller"},
		{subcomponentName: "capi-ocne", imageName: "cluster-api-ocne-control-plane-controller"},
	},
}

var runner vzos.CmdRunner = vzos.DefaultRunner{}

// TestNewCatalogModuleVersions makes sure the module versions in the catalog are up-to-date with the Bom
// GIVEN the module catalog
// ENSURE that each module's version is up to date with the bom version
func TestNewCatalogModuleVersions(t *testing.T) {
	config.SetDefaultBomFilePath(bomPath)

	catalog, err := NewCatalog(catalogPath)
	assert.NoError(t, err)
	assert.NotNil(t, catalog)

	vzBOM, err := bom.NewBom(bomPath)
	assert.NoError(t, err)
	assert.NotNil(t, vzBOM)

	var bomVersion, moduleVersion string
	for _, module := range catalog.Modules {
		if vzString.SliceContainsString(modulesNotInBom, module.Name) {
			continue
		}
		// check to see if this is a module without a top level BOM component
		subcomponent, ok := subcomponentOverrides[module.Name]
		if ok {
			image, err := vzBOM.FindImage(subcomponent[0].subcomponentName, subcomponent[0].imageName)
			assert.NoError(t, err)
			imageTagSemver, err := semver.NewSemVersion(image.ImageTag)
			assert.NoError(t, err)
			assert.NotNil(t, imageTagSemver)
			bomVersion = imageTagSemver.ToStringWithoutBuildAndPrerelease()
			moduleVersionSemver, err := semver.NewSemVersion(module.Version)
			assert.NoError(t, err)
			moduleVersion = moduleVersionSemver.ToStringWithoutBuildAndPrerelease()
		} else {
			bomVersion, err = vzBOM.GetComponentVersion(module.Name)
			assert.NoError(t, err)
			moduleVersion = module.Version
		}
		assert.Equalf(t, bomVersion, moduleVersion,
			"Catalog module version and BOM version for module %s out of date, BOM version: %s, catalog version %s",
			module.Name, bomVersion, module.Version)
	}
}

// TestNewCatalogModuleVersionsTestBOM makes sure the internal components are being set properly from the BOM
// GIVEN a fake BOM
// ENSURE the app operator has the expected version
func TestNewCatalogModuleVersionsTestBOM(t *testing.T) {
	config.SetDefaultBomFilePath(testBOMPath)

	catalog, err := NewCatalog(catalogPath)
	assert.NoError(t, err)
	assert.NotNil(t, catalog)

	assert.Equalf(t, "1.1.0", catalog.GetVersion(appoper.ComponentName),
		"Catalog entry for module verrazzno-application-operator is incorrect")
}

// TestGetVersionForAllRegistryComponents compares the catalog and the component registry
// GIVEN the module catalog
// ENSURE that each module has a corresponding registry component
func TestGetVersionForAllRegistryComponents(t *testing.T) {
	config.SetDefaultBomFilePath(bomPath)

	catalog, err := NewCatalog(catalogPath)
	assert.NoError(t, err)
	assert.NotNil(t, catalog)

	for _, component := range registry.GetComponents() {
		assert.NotNilf(t, catalog.GetVersion(component.Name()), "Failed to find matching catalog version for components %s", component.Name())
	}
}

// TestCompareBOMWithRemote ensures that if the BOM on the feature branch has been updated,
// the corresponding catalog module version has also been upgraded
// GIVEN the BOM and catalog from the branch and the BOM and catalog from the target branch
// IF the BOM on this branch has been updated since the common ancestor commit with the target branch
// ENSURE that if the BOM entry for a module has been updated, the module version has also been updated
func TestCompareBOMWithRemote(t *testing.T) {
	// check if the BOM on this branch has been updated since the common ancestor commit with the target branch
	// don't run check if not
	// this is so that merges to the BOM on the target branch don't fail this test on other feature branches
	if slices.Contains(getTargetBranchDiff(t), "platform-operator/verrazzano-bom.json") {
		config.SetDefaultBomFilePath(bomPath)

		// get the local bom
		localBOM, err := bom.NewBom(bomPath)
		assert.NoError(t, err)
		assert.NotNil(t, localBOM)

		// get the remote bom from the target branch
		remoteBOM := getRemoteBOM(t)

		// get the local catalog
		localCatalog, err := NewCatalog(catalogPath)
		assert.NoError(t, err)
		assert.NotNil(t, localCatalog)

		// get the remote catalog from the target branch
		remoteCatalog := getRemoteCatalog(t)

		for _, module := range localCatalog.Modules {
			// if this is a new module that doesn't exist in the remote catalog, continue
			if remoteCatalog.GetVersion(module.Name) == "" {
				continue
			}
			// if the BOM entry for the module has been updated ensure the module version has been updated as well
			if checkIfModuleUpdated(t, module, localBOM, remoteBOM) {
				localVersion, err := semver.NewSemVersion(localCatalog.GetVersion(module.Name))
				assert.NoError(t, err)
				remoteVersion, err := semver.NewSemVersion(remoteCatalog.GetVersion(module.Name))
				assert.NoError(t, err)
				compare, err := localVersion.CompareToPrereleaseInts(remoteVersion)
				assert.NoErrorf(t, err, "Unexpected error comparing prerelease fields for module %s.\n"+
					"Version on local branch: %s, Version on target branch %s: %s",
					module.Name, localVersion.ToString(), targetBranch, remoteVersion.ToString())
				assert.Equalf(t, compare, 1,
					"BOM entry for module %s on this branch has been modified from the one on %s.\n"+
						"The catalog module version %s must also be modifed to be greater than remote catalog module version %s on target branch %s.\n"+
						"Increment the prerelease version for module %s in the catalog (platform-operator/manifests/catalog/catalog.yaml) "+
						"and update the corresponding BOM component version.",
					module.Name, targetBranch, localVersion.ToString(), remoteVersion.ToString(), targetBranch, module.Name)
			}
		}
	}
}

// TestCompareChartsWithRemote ensures that if any of the chart or values overrides files on the feature branch
// has been updated, the corresponding catalog module version has also been upgraded
// GIVEN a list of all the files updated on a target branch
// IF any of the chart or values overrides files have been updated since the common ancestor commit with the target branch
// ENSURE that the corresponding module version is also greater than the version on the target branch
func TestCompareChartsWithRemote(t *testing.T) {
	config.SetDefaultBomFilePath(bomPath)

	// get the local catalog
	localCatalog, err := NewCatalog(catalogPath)
	assert.NoError(t, err)
	assert.NotNil(t, localCatalog)

	// get the remote catalog from the target branch
	remoteCatalog := getRemoteCatalog(t)

	// generate the file diff against the target branch
	changes := getTargetBranchDiff(t)

	for _, module := range localCatalog.Modules {
		// if this is a new module that doesn't exist in the remote catalog, continue
		if remoteCatalog.GetVersion(module.Name) == "" ||
			module.Version == constants.BomVerrazzanoVersion {
			continue
		}
		var changedFiles []string
		for _, change := range changes {
			if (module.Chart != "" && strings.HasPrefix(change, module.Chart)) ||
				(len(module.ValuesFiles) > 0 && slices.Contains(module.ValuesFiles, change)) {
				changedFiles = append(changedFiles, change)
			}
		}
		if len(changedFiles) != 0 {
			localVersion, err := semver.NewSemVersion(localCatalog.GetVersion(module.Name))
			assert.NoError(t, err)
			remoteVersion, err := semver.NewSemVersion(remoteCatalog.GetVersion(module.Name))
			assert.NoError(t, err)
			compare, err := localVersion.CompareToPrereleaseInts(remoteVersion)
			assert.NoErrorf(t, err, "Unexpected error comparing prerelease fields for module %s.\n"+
				"Version on local branch: %s, Version on target branch %s: %s",
				module.Name, localVersion.ToString(), targetBranch, remoteVersion.ToString())
			assert.Equalf(t, compare, 1,
				"The following chart or values overrides files for module %s on this branch has been modified from the files on %s:\n%v\n"+
					"The catalog module version %s must be modifed to be greater than remote catalog module version %s on target branch %s.\n"+
					"Increment the prerelease version for module %s in the catalog (platform-operator/manifests/catalog/catalog.yaml) "+
					"and update the corresponding BOM component version.",
				module.Name, targetBranch, changedFiles, localVersion.ToString(), remoteVersion.ToString(), targetBranch, module.Name)
		}
	}
}

func getTargetBranchDiff(t *testing.T) []string {
	arg := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", targetBranch, targetBranch)
	_, err := exec.Command("git", "config", "--add", "remote.origin.fetch", arg).Output()
	assert.NoError(t, err)
	_, err = exec.Command("git", "fetch").Output()
	assert.NoError(t, err)
	cmd := exec.Command("git", "diff", "--name-only", fmt.Sprintf("remotes/origin/%s...", targetBranch)) // #nosec G204
	stdout, stderr, err := runner.Run(cmd)
	assert.NoError(t, err)
	assert.Emptyf(t, err, "StdErr should be empty: %s", string(stderr))
	return strings.Split(string(stdout), "\n")
}

func checkIfModuleUpdated(t *testing.T, module Module, localBOM, remoteBOM bom.Bom) bool {
	// skip module if:
	//	- it is a module that doesn't have a bom version
	//	- it is a module that is versioned with the BOM
	//	- it is a new module that doesn't exist yet in the remote catalog
	if vzString.SliceContainsString(modulesNotInBom, module.Name) ||
		module.Version == constants.BomVerrazzanoVersion {
		return false
	}
	// check to see if this is a module without a top level BOM component
	overridedModule, ok := subcomponentOverrides[module.Name]
	if ok {
		for _, override := range overridedModule {
			localBOMImage, err := localBOM.FindImage(override.subcomponentName, override.imageName)
			assert.NoError(t, err)
			remoteBOMImage, err := remoteBOM.FindImage(override.subcomponentName, override.imageName)
			assert.NoError(t, err)
			if localBOMImage != remoteBOMImage {
				return true
			}
		}
	} else {
		localComponent, err := localBOM.GetComponent(module.Name)
		assert.NoError(t, err)
		remoteComponent, err := remoteBOM.GetComponent(module.Name)
		assert.NoError(t, err)
		if !reflect.DeepEqual(localComponent, remoteComponent) {
			return true
		}
	}
	return false
}

func getRemoteCatalog(t *testing.T) Catalog {
	req, err := retryablehttp.NewRequest("GET", remoteCatalogPath, nil)
	assert.NoError(t, err)
	client := retryablehttp.NewClient()
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	bodyRaw, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NotEmpty(t, bodyRaw)
	remoteCatalog, err := NewCatalogfromYAML(bodyRaw)
	assert.NoError(t, err)
	assert.NotNil(t, remoteCatalog)
	return remoteCatalog
}

func getRemoteBOM(t *testing.T) bom.Bom {
	req, err := retryablehttp.NewRequest("GET", remoteBOMPath, nil)
	assert.NoError(t, err)
	client := retryablehttp.NewClient()
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	bodyRaw, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NotEmpty(t, bodyRaw)
	remoteBOM, err := bom.NewBOMFromJSON(bodyRaw)
	assert.NoError(t, err)
	assert.NotNil(t, remoteBOM)
	return remoteBOM
}
