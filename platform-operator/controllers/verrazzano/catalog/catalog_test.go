// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package catalog

import (
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"io"
	"os"
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
const remoteBOMPath = "https://raw.githubusercontent.com/verrazzano/verrazzano/master/platform-operator/verrazzano-bom.json"
const remoteCatalogPath = "https://raw.githubusercontent.com/verrazzano/verrazzano/master/platform-operator/manifests/catalog/catalog.yaml"

type bomSubcomponentOverrides struct {
	subcomponentName string
	imageName        string
}

var modulesNotInBom = []string{
	"verrazzano-grafana-dashboards",
	"verrazzano-network-policies",
	"cluster-issuer",
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
			"Catalog entry for module %s out of date, BOM version: %s, catalog version %s", module.Name, bomVersion, module.Version)
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
// IF the BOM on this branch has been updated since the common ancestor commit with the target branch
// GIVEN the BOM and catalog from the branch and the BOM and catalog from master
// ENSURE that if the BOM entry for a module has been updated, the module version has also been updated
func TestCompareBOMWithRemote(t *testing.T) {
	// check if the BOM on this branch has been updated since the common ancestor commit with the target branch
	// don't run check if not
	// this is so that merges to the BOM in master don't fail this test on other feature branches
	if checkBOMModifiedInBranch(t) {
		config.SetDefaultBomFilePath(bomPath)

		// get the local bom
		localBOM, err := bom.NewBom(bomPath)
		assert.NoError(t, err)
		assert.NotNil(t, localBOM)

		// get the remote bom from master
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

		// get the local catalog
		localCatalog, err := NewCatalog(catalogPath)
		assert.NoError(t, err)
		assert.NotNil(t, localCatalog)

		// get the remote catalog from master
		req, err = retryablehttp.NewRequest("GET", remoteCatalogPath, nil)
		assert.NoError(t, err)
		resp, err = client.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		bodyRaw, err = io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.NotEmpty(t, bodyRaw)
		remoteCatalog, err := NewCatalogfromYAML(bodyRaw)
		assert.NoError(t, err)
		assert.NotNil(t, remoteCatalog)

		for _, module := range localCatalog.Modules {
			if vzString.SliceContainsString(modulesNotInBom, module.Name) ||
				module.Version == constants.BomVerrazzanoVersion {
				continue
			}
			// check to see if this is a module without a top level BOM component
			overridedModule, ok := subcomponentOverrides[module.Name]
			updated := false
			if ok {
				for _, override := range overridedModule {
					localBOMImage, err := localBOM.FindImage(override.subcomponentName, override.imageName)
					assert.NoError(t, err)
					remoteBOMImage, err := remoteBOM.FindImage(override.subcomponentName, override.imageName)
					assert.NoError(t, err)
					if localBOMImage != remoteBOMImage {
						updated = true
					}
				}
			} else {
				localComponent, err := localBOM.GetComponent(module.Name)
				assert.NoError(t, err)
				remoteComponent, err := remoteBOM.GetComponent(module.Name)
				assert.NoError(t, err)
				if !reflect.DeepEqual(localComponent, remoteComponent) {
					updated = true
				}
			}

			// bom entry for the module has been updated, so ensure the module version has been updated as well
			if updated {
				localVersion, err := semver.NewSemVersion(localCatalog.GetVersion(module.Name))
				assert.NoError(t, err)
				remoteVersion, err := semver.NewSemVersion(remoteCatalog.GetVersion(module.Name))
				assert.NoError(t, err)
				assert.Truef(t, localVersion.IsGreatherThan(remoteVersion),
					"BOM entry for module %s has been updated so local catalog version %s should be greater than remote catalog version %s",
					module.Name, localVersion.ToString(), remoteVersion.ToString())
			}
		}
	}
}

func checkBOMModifiedInBranch(t *testing.T) bool {
	var out []byte
	_, err := exec.Command("git", "fetch", "origin").Output()
	assert.NoError(t, err)

	cmd := exec.Command("git", "diff", "--name-only", "remotes/origin/master...remotes/origin/"+os.Getenv("BRANCH_NAME"))
	assert.NoError(t, err)
	stdout, stderr, err := runner.Run(cmd)
	assert.NoErrorf(t, err, "error: %s, %s", string(stdout), string(stderr))
	return strings.Contains(string(out), "platform-operator/verrazzano-bom.json")
}
