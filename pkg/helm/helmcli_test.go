// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
)

const ns = "my_namespace"
const chartdir = "my_charts"
const release = "my_release"
const missingRelease = "no_release"

// upgradeRunner is used to test Helm upgrade without actually running an OS exec command
type upgradeRunner struct {
	t *testing.T

	expectedArgs []string
}

// getValuesRunner is used to test Helm get values without actually running an OS exec command
type getValuesRunner struct {
	t *testing.T
}

// badRunner is used to test Helm errors without actually running an OS exec command
type badRunner struct {
	t *testing.T
}

// foundRunner is used to test helm status command
type foundRunner struct {
	t *testing.T
}

// genericTestRunner is used to run generic OS commands with expected results
type genericTestRunner struct {
	stdOut []byte
	stdErr []byte
	err    error
}

// Run genericTestRunner executor
func (r genericTestRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return r.stdOut, r.stdErr, r.err
}

// TestGetValues tests the Helm get values command
// GIVEN a set of upgrade parameters
//
//	WHEN I call Upgrade
//	THEN the Helm upgrade returns success and the cmd object has correct values
func TestGetValues(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(getValuesRunner{t: t})
	defer SetDefaultRunner()

	stdout, err := GetValues(vzlog.DefaultLogger(), release, ns)
	assert.NoError(err, "GetValues returned an error")
	assert.NotZero(stdout, "GetValues stdout should not be empty")
}

// TestUpgrade tests the Helm upgrade command
// GIVEN a set of upgrade parameters
//
//	WHEN I call Upgrade
//	THEN the Helm upgrade returns success and the cmd object has correct values
func TestUpgrade(t *testing.T) {
	var overrides []HelmOverrides
	overrides = append(overrides, HelmOverrides{FileOverride: "my-override.yaml"})
	assert := assert.New(t)
	SetCmdRunner(upgradeRunner{
		t: t,
		expectedArgs: []string{
			"--namespace",
			"my_namespace",
			"--install",
			"-f",
			"my-override.yaml",
		},
	})
	defer SetDefaultRunner()

	stdout, stderr, err := Upgrade(vzlog.DefaultLogger(), release, ns, chartdir, false, false, overrides)
	assert.NoError(err, "Upgrade returned an error")
	assert.Len(stderr, 0, "Upgrade stderr should be empty")
	assert.NotZero(stdout, "Upgrade stdout should not be empty")
}

// TestUpgradeCustomFileOverrides tests the Helm upgrade command
// GIVEN a set of upgrade parameters with additional file overrides
//
//	WHEN I call Upgrade
//	THEN the Helm upgrade returns success and the cmd object has correct values
func TestUpgradeCustomFileOverrides(t *testing.T) {
	var overrides []HelmOverrides
	overrides = append(overrides, HelmOverrides{FileOverride: "my-override.yaml"})
	overrides = append(overrides, HelmOverrides{FileOverride: "custom-override.yaml"})
	assert := assert.New(t)
	SetCmdRunner(upgradeRunner{
		t: t,
		expectedArgs: []string{
			"--namespace",
			"my_namespace",
			"--install",
			"-f",
			"my-override.yaml",
			"-f",
			"custom-override.yaml",
		},
	})
	defer SetDefaultRunner()

	stdout, stderr, err := Upgrade(vzlog.DefaultLogger(), release, ns, chartdir, false, false, overrides)
	assert.NoError(err, "Upgrade returned an error")
	assert.Len(stderr, 0, "Upgrade stderr should be empty")
	assert.NotZero(stdout, "Upgrade stdout should not be empty")
}

// TestUpgradeFail tests the Helm upgrade command failure condition
// GIVEN a set of upgrade parameters and a fake runner that fails
//
//	WHEN I call Upgrade
//	THEN the Helm upgrade returns an error
func TestUpgradeFail(t *testing.T) {
	var overrides []HelmOverrides
	assert := assert.New(t)
	SetCmdRunner(badRunner{t: t})
	defer SetDefaultRunner()

	stdout, stderr, err := Upgrade(vzlog.DefaultLogger(), release, ns, "", false, false, overrides)
	assert.Error(err, "Upgrade should have returned an error")
	assert.Len(stdout, 0, "Upgrade stdout should be empty")
	assert.NotZero(stderr, "Upgrade stderr should not be empty")
}

// TestUninstall tests the Helm Uninstall fn
// GIVEN a call to Uninstall
//
//	WHEN the command executes successfully
//	THEN the function returns no error
func TestUninstall(t *testing.T) {
	stdout := []byte{}
	stdErr := []byte{}

	SetCmdRunner(genericTestRunner{
		stdOut: stdout,
		stdErr: stdErr,
		err:    nil,
	})
	defer SetDefaultRunner()
	_, _, err := Uninstall(vzlog.DefaultLogger(), "weblogic-operator", "verrazzano-system", false)
	assert.NoError(t, err)
}

// TestUninstallError tests the Helm Uninstall fn
// GIVEN a call to Uninstall
//
//	WHEN the command executes and returns an error
//	THEN the function returns an error
func TestUninstallError(t *testing.T) {
	var stdout []byte
	var stdErr []byte

	SetCmdRunner(genericTestRunner{
		stdOut: stdout,
		stdErr: stdErr,
		err:    fmt.Errorf("Unexpected uninstall error"),
	})
	defer SetDefaultRunner()
	_, _, err := Uninstall(vzlog.DefaultLogger(), "weblogic-operator", "verrazzano-system", false)
	assert.Error(t, err)
}

// TestIsReleaseInstalled tests checking if a Helm release is installed
// GIVEN a release name and namespace
//
//	WHEN I call IsReleaseInstalled
//	THEN the function returns success and found equal true
func TestIsReleaseInstalled(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(foundRunner{t: t})
	defer SetDefaultRunner()

	found, err := IsReleaseInstalled(release, ns)
	assert.NoError(err, "IsReleaseInstalled returned an error")
	assert.True(found, "Release not found")
}

// TestIsReleaseNotInstalled tests checking if a Helm release is not installed
// GIVEN a release name and namespace
//
//	WHEN I call IsReleaseInstalled
//	THEN the function returns success and the correct found status
func TestIsReleaseNotInstalled(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(foundRunner{t: t})
	defer SetDefaultRunner()

	found, err := IsReleaseInstalled(missingRelease, ns)
	assert.NoError(err, "IsReleaseInstalled returned an error")
	assert.False(found, "Release should not be found")
}

// TestIsReleaseInstalledFailed tests failure when checking if a Helm release is installed
// GIVEN a bad release name and namespace
//
//	WHEN I call IsReleaseInstalled
//	THEN the function returns a failure
func TestIsReleaseInstalledFailed(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(foundRunner{t: t})
	defer SetDefaultRunner()

	found, err := IsReleaseInstalled("", ns)
	assert.Error(err, "IsReleaseInstalled should have returned an error")
	assert.False(found, "Release should not be found")
}

// TestIsReleaseDeployed tests checking if a Helm release is installed
// GIVEN a release name and namespace
//
//	WHEN I call IsReleaseDeployed
//	THEN the function returns success and found equal true
func TestIsReleaseDeployed(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(foundRunner{t: t})
	defer SetDefaultRunner()
	SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return ChartStatusDeployed, nil
	})
	defer SetDefaultChartStatusFunction()

	found, err := IsReleaseDeployed(release, ns)
	assert.NoError(err, "IsReleaseInstalled returned an error")
	assert.True(found, "Release not found")
}

// TestIsReleaseNotDeployed tests checking if a Helm release is not installed
// GIVEN a release name and namespace
//
//	WHEN I call IsReleaseDeployed
//	THEN the function returns success and the correct found status
func TestIsReleaseNotDeployed(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(foundRunner{t: t})
	defer SetDefaultRunner()
	SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return ChartNotFound, nil
	})
	defer SetDefaultChartStatusFunction()

	found, err := IsReleaseDeployed(missingRelease, ns)
	assert.NoError(err, "IsReleaseInstalled returned an error")
	assert.False(found, "Release should not be found")
}

// TestIsReleaseFailedChartNotFound tests checking if a Helm release is in a failed state
// GIVEN a release name and namespace
//
//	WHEN I call IsReleaseFailed and the status is ChartNotFound
//	THEN the function returns false and no error
func TestIsReleaseFailedChartNotFound(t *testing.T) {
	assert := assert.New(t)
	SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return ChartNotFound, nil
	})
	defer SetDefaultChartStateFunction()

	failed, err := IsReleaseFailed("foo", "bar")
	assert.NoError(err, "IsReleaseInstalled returned an error")
	assert.False(failed, "ReleaseFailed should be false")
}

// TestIsReleaseFailedChartNotFound tests checking if a Helm release is in a failed state
// GIVEN a release name and namespace
//
//	WHEN I call IsReleaseFailed and the status is deployed
//	THEN the function returns false and no error
func TestIsReleaseFailedChartDeployed(t *testing.T) {
	assert := assert.New(t)
	SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return ChartStatusDeployed, nil
	})
	defer SetDefaultChartStateFunction()

	failed, err := IsReleaseFailed("foo", "bar")
	assert.NoError(err, "IsReleaseInstalled returned an error")
	assert.False(failed, "ReleaseFailed should be false")
}

// TestIsReleaseFailed tests checking if a Helm release is in a failed state
// GIVEN a release name and namespace
//
//	WHEN I call IsReleaseFailed and the status is failed
//	THEN the function returns false and no error
func TestIsReleaseFailed(t *testing.T) {
	assert := assert.New(t)
	SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return ChartStatusFailed, nil
	})
	defer SetDefaultChartStateFunction()

	failed, err := IsReleaseFailed("foo", "bar")
	assert.NoError(err, "IsReleaseInstalled returned an error")
	assert.True(failed, "ReleaseFailed should be true")
}

// TestIsReleaseFailedError tests checking if a Helm release is in a failed state
// GIVEN a release name and namespace
//
//	WHEN I call IsReleaseFailed and the an error is returned by the state function
//	THEN the function returns false and an error
func TestIsReleaseFailedError(t *testing.T) {
	assert := assert.New(t)
	SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return "", fmt.Errorf("Unexpected error")
	})
	defer SetDefaultChartStateFunction()

	failed, err := IsReleaseFailed("foo", "bar")
	assert.Error(err, "IsReleaseInstalled returned an error")
	assert.False(failed)
}

// Test_getReleaseStateDeployed tests the getReleaseState fn
// GIVEN a call to getReleaseState
//
//	WHEN the chart state is deployed
//	THEN the function returns ChartStatusDeployed and no error
func Test_getReleaseStateDeployed(t *testing.T) {
	jsonOut := []byte(`
[
  {
    "name": "weblogic-operator",
    "namespace": "verrazzano-system",
    "revision": "1",
    "updated": "2021-09-08 17:15:01.516374225 +0000 UTC",
    "status": "deployed",
    "chart": "weblogic-operator-3.3.0",
    "app_version": "3.3.0"
  }
]
`)

	SetCmdRunner(genericTestRunner{
		stdOut: jsonOut,
		stdErr: []byte{},
		err:    nil,
	})
	defer SetDefaultRunner()
	state, err := getReleaseState("weblogic-operator", "verrazzano-system")
	assert.NoError(t, err)
	assert.Equalf(t, ChartStatusDeployed, state, "unpexected state: %s", state)
}

// Test_getReleaseStateDeployed tests the getReleaseState fn
// GIVEN a call to getReleaseState
//
//	WHEN the chart state is pending-install
//	THEN the function returns ChartStatusPendingInstall and no error
func Test_getReleaseStatePendingInstall(t *testing.T) {
	jsonOut := []byte(`
[
  {
    "name": "weblogic-operator",
    "namespace": "verrazzano-system",
    "revision": "1",
    "updated": "2021-09-08 17:15:01.516374225 +0000 UTC",
    "status": "pending-install",
    "chart": "weblogic-operator-3.3.0",
    "app_version": "3.3.0"
  }
]
`)

	SetCmdRunner(genericTestRunner{
		stdOut: jsonOut,
		stdErr: []byte{},
		err:    nil,
	})
	defer SetDefaultRunner()
	state, err := getReleaseState("weblogic-operator", "verrazzano-system")
	assert.NoError(t, err)
	assert.Equalf(t, ChartStatusPendingInstall, state, "unpexected state: %s", state)
}

// Test_getReleaseStateChartNotFound tests the getReleaseState fn
// GIVEN a call to getReleaseState
//
//	WHEN the chart/release can not be found
//	THEN the function returns "" and no error
func Test_getReleaseStateChartNotFound(t *testing.T) {
	jsonOut := []byte(`[]`)

	SetCmdRunner(genericTestRunner{
		stdOut: jsonOut,
		stdErr: []byte{},
		err:    nil,
	})
	defer SetDefaultRunner()
	state, err := getReleaseState("weblogic-operator", "verrazzano-system")
	assert.NoError(t, err)
	assert.Equalf(t, "", state, "unpexected state: %s", state)
}

// Test_getChartStatusDeployed tests the getChartStatus fn
// GIVEN a call to getChartStatus
//
//	WHEN Helm returns a deployed state
//	THEN the function returns "deployed" and no error
func Test_getChartStatusDeployed(t *testing.T) {
	jsonOut := []byte(`
{
  "name": "weblogic-operator",
  "info": {
    "first_deployed": "2021-09-08T17:15:01.516374225Z",
    "last_deployed": "2021-09-08T17:15:01.516374225Z",
    "deleted": "",
    "description": "Install complete",
    "status": "deployed"
  },
  "config": {
    "annotations": {
      "traffic.sidecar.istio.io/excludeOutboundPorts": "443"
    },
    "domainNamespaceLabelSelector": "verrazzano-managed",
    "domainNamespaceSelectionStrategy": "LabelSelector",
    "enableClusterRoleBinding": true,
    "image": "ghcr.io/oracle/weblogic-kubernetes-operator:3.3.0",
    "imagePullSecrets": [
      {
        "name": "verrazzano-container-registry"
      }
    ],
    "serviceAccount": "weblogic-operator-sa"
  },
  "manifest": "manifest-text",
  "version": 1,
  "namespace": "verrazzano-system"
}
`)

	SetCmdRunner(genericTestRunner{
		stdOut: jsonOut,
		stdErr: []byte{},
		err:    nil,
	})
	defer SetDefaultRunner()
	state, err := getChartStatus("weblogic-operator", "verrazzano-system")
	assert.NoError(t, err)
	assert.Equalf(t, ChartStatusDeployed, state, "unpexected state: %s", state)
}

// Test_getChartStatusNotFound tests the getChartStatus fn
// GIVEN a call to getChartStatus
//
//	WHEN the json structure does not have an status filed in the info section
//	THEN the function returns an error
func Test_getChartStatusNotFound(t *testing.T) {
	jsonOut := []byte(`
{
  "name": "weblogic-operator",
  "info": {
    "first_deployed": "2021-09-08T17:15:01.516374225Z",
    "last_deployed": "2021-09-08T17:15:01.516374225Z",
    "deleted": "",
    "description": "Install complete",
  },
  "config": {
    "annotations": {
      "traffic.sidecar.istio.io/excludeOutboundPorts": "443"
    },
    "domainNamespaceLabelSelector": "verrazzano-managed",
    "domainNamespaceSelectionStrategy": "LabelSelector",
    "enableClusterRoleBinding": true,
    "image": "ghcr.io/oracle/weblogic-kubernetes-operator:3.3.0",
    "imagePullSecrets": [
      {
        "name": "verrazzano-container-registry"
      }
    ],
    "serviceAccount": "weblogic-operator-sa"
  },
  "manifest": "manifest-text",
  "version": 1,
  "namespace": "verrazzano-system"
}
`)

	SetCmdRunner(genericTestRunner{
		stdOut: jsonOut,
		stdErr: []byte{},
		err:    nil,
	})
	defer SetDefaultRunner()
	state, err := getChartStatus("weblogic-operator", "verrazzano-system")
	assert.Error(t, err)
	assert.Empty(t, state)
}

// Test_getChartStatusChartNotFound tests the getChartStatus fn
// GIVEN a call to getChartStatus
//
//	WHEN the Chart is not found
//	THEN the function returns chart not found and no error
func Test_getChartStatusChartNotFound(t *testing.T) {
	stdout := []byte{}
	stdErr := []byte("Error: release: not found")

	SetCmdRunner(genericTestRunner{
		stdOut: stdout,
		stdErr: stdErr,
		err:    fmt.Errorf("Error running status command"),
	})
	defer SetDefaultRunner()
	state, err := getChartStatus("weblogic-operator", "verrazzano-system")
	assert.NoError(t, err)
	assert.Equalf(t, ChartNotFound, state, "unpexected state: %s", state)
}

// Test_getChartStatusUnexpectedHelmError tests the getChartStatus fn
// GIVEN a call to getChartStatus
//
//	WHEN Helm returns an error
//	THEN the function returns an error
func Test_getChartStatusUnexpectedHelmError(t *testing.T) {
	var stdout []byte
	stdErr := []byte("Some unknown helm status error")

	SetCmdRunner(genericTestRunner{
		stdOut: stdout,
		stdErr: stdErr,
		err:    fmt.Errorf("Unexpected error running status command"),
	})
	defer SetDefaultRunner()
	state, err := getChartStatus("weblogic-operator", "verrazzano-system")
	assert.Error(t, err)
	assert.Equalf(t, "", state, "unpexected state: %s", state)
}

// Test_getChartInfoNotFound tests the getChartStatus fn
// GIVEN a call to getChartStatus
//
//	WHEN the json structure does not have an info section
//	THEN the function returns an error
func Test_getChartInfoNotFound(t *testing.T) {
	jsonOut := []byte(`
{
  "name": "weblogic-operator",
  "config": {
    "annotations": {
      "traffic.sidecar.istio.io/excludeOutboundPorts": "443"
    },
    "domainNamespaceLabelSelector": "verrazzano-managed",
    "domainNamespaceSelectionStrategy": "LabelSelector",
    "enableClusterRoleBinding": true,
    "image": "ghcr.io/oracle/weblogic-kubernetes-operator:3.3.0",
    "imagePullSecrets": [
      {
        "name": "verrazzano-container-registry"
      }
    ],
    "serviceAccount": "weblogic-operator-sa"
  },
  "manifest": "manifest-text",
  "version": 1,
  "namespace": "verrazzano-system"
}
`)

	SetCmdRunner(genericTestRunner{
		stdOut: jsonOut,
		stdErr: []byte{},
		err:    nil,
	})
	defer SetDefaultRunner()
	state, err := getChartStatus("weblogic-operator", "verrazzano-system")
	assert.Error(t, err)
	assert.Empty(t, state)
}

// TestGetReleaseValue tests the GetReleaseValues fn
// GIVEN a call to GetReleaseValues
//
//	WHEN a valid helm release and namespace are deployed
//	THEN the function returns the value/true/nil if the helm key exists, or ""/false/nil if it doesn't
func TestGetReleaseValue(t *testing.T) {
	jsonOut := []byte(`
{
  "domainFilters": [
    "my.domain.io"
  ],
  "triggerLoopOnEvent": true,
  "txtOwnerId": "v8o-default-my-verrazzano-3201314693",
  "txtPrefix": "_v8o-default-my-verrazzano-3201314693-",
  "zoneIDFilters": [
    "ocid1.dns-zone.oc1..blahblahblah"
  ]
}
`)

	SetCmdRunner(genericTestRunner{
		stdOut: jsonOut,
		stdErr: []byte{},
		err:    nil,
	})
	defer SetDefaultRunner()

	SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return ChartStatusDeployed, nil
	})
	defer SetDefaultChartStatusFunction()

	keys := []string{"txtOwnerId", "zoneIDFilters", "foo"}
	value, err := GetReleaseValues(vzlog.DefaultLogger(), keys, "external-dns", "cert-manager")

	expectedMap := map[string]interface{}{
		"txtOwnerId":    "v8o-default-my-verrazzano-3201314693",
		"zoneIDFilters": []interface{}{"ocid1.dns-zone.oc1..blahblahblah"},
	}
	assert.NoError(t, err)
	assert.Equal(t, expectedMap, value, "Map did not contain expected values")
}

// TestGetReleaseStringValue tests the GetReleaseStringValues fn
// GIVEN a call to GetReleaseStringValues
//
//	WHEN a valid helm release and namespace are deployed
//	THEN the function returns the value/true/nil if the helm key exists, or ""/false/nil if it doesn't
func TestGetReleaseStringValue(t *testing.T) {
	jsonOut := []byte(`
{
  "domainFilters": [
    "my.domain.io"
  ],
  "triggerLoopOnEvent": true,
  "txtOwnerId": "v8o-default-my-verrazzano-3201314693",
  "txtPrefix": "_v8o-default-my-verrazzano-3201314693-",
  "zoneIDFilters": [
    "ocid1.dns-zone.oc1..blahblahblah"
  ]
}
`)

	SetCmdRunner(genericTestRunner{
		stdOut: jsonOut,
		stdErr: []byte{},
		err:    nil,
	})
	defer SetDefaultRunner()

	SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return ChartStatusDeployed, nil
	})
	defer SetDefaultChartStatusFunction()

	keys := []string{"txtOwnerId", "zoneIDFilters", "foo"}
	value, err := GetReleaseStringValues(vzlog.DefaultLogger(), keys, "external-dns", "cert-manager")

	expectedMap := map[string]string{
		"txtOwnerId":    "v8o-default-my-verrazzano-3201314693",
		"zoneIDFilters": "[ocid1.dns-zone.oc1..blahblahblah]",
	}
	assert.NoError(t, err)
	assert.Equal(t, expectedMap, value, "Map did not contain expected values")
}

// TestGetReleaseValueReleaseNotFound tests the GetReleaseValues fn
// GIVEN a call to GetReleaseValues
//
//	WHEN a the helm release is NOT deployed
//	THEN the function returns the value/true/nil if the helm key exists, or ""/false/nil if it doesn't
func TestGetReleaseValueReleaseNotFound(t *testing.T) {

	SetCmdRunner(genericTestRunner{
		stdOut: []byte{},
		stdErr: []byte("Chart not found"),
		err:    nil,
	})
	defer SetDefaultRunner()

	SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return ChartNotFound, nil
	})
	defer SetDefaultChartStatusFunction()

	keys := []string{"txtOwnerId", "external-dns"}
	values, err := GetReleaseValues(vzlog.DefaultLogger(), keys, "external-dns", "cert-manager")
	assert.NoErrorf(t, err, "Unexpected error: %v", err)
	assert.Equal(t, map[string]interface{}{}, values, "Found unexpected release value")

	expectedErr := fmt.Errorf("Helm error")
	SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return ChartNotFound, expectedErr
	})
	values, helmErr := GetReleaseValues(vzlog.DefaultLogger(), keys, "external-dns", "cert-manager")
	assert.NoErrorf(t, err, "Unexpected error: %v", err)
	assert.Equal(t, map[string]interface{}{}, values, "Found unexpected release values")
	assert.Error(t, helmErr, "Did not get expected error")
	assert.Equal(t, expectedErr, helmErr)
}

// Test_maskSensitiveData tests the maskSensitiveData function
func Test_maskSensitiveData(t *testing.T) {
	// GIVEN a string with sensitive data
	// WHEN the maskSensitiveData function is called
	// THEN the returned string has sensitive values masked
	str := `Running command: /usr/bin/helm upgrade mysql /verrazzano/platform-operator/thirdparty/charts/mysql
		--wait --namespace keycloak --install -f /verrazzano/platform-operator/helm_config/overrides/mysql-values.yaml
		-f /tmp/values-145495151.yaml
		--set imageTag=8.0.26,image=ghcr.io/verrazzano/mysql,mysqlPassword=BgD2SBNaGm,mysqlRootPassword=ydqtBpasQ4`
	expected := `Running command: /usr/bin/helm upgrade mysql /verrazzano/platform-operator/thirdparty/charts/mysql
		--wait --namespace keycloak --install -f /verrazzano/platform-operator/helm_config/overrides/mysql-values.yaml
		-f /tmp/values-145495151.yaml
		--set imageTag=8.0.26,image=ghcr.io/verrazzano/mysql,mysqlPassword=*****,mysqlRootPassword=*****`
	maskedStr := maskSensitiveData(str)
	assert.Equal(t, expected, maskedStr)

	// GIVEN a string without sensitive data
	// WHEN the maskSensitiveData function is called
	// THEN the returned string is unaltered
	str = `Running command: /usr/bin/helm upgrade ingress-controller /verrazzano/platform-operator/thirdparty/charts/ingress-nginx
		--wait --namespace ingress-nginx --install -f /verrazzano/platform-operator/helm_config/overrides/ingress-nginx-values.yaml
		-f /tmp/values-037653479.yaml --set controller.image.tag=0.46.0-20211005200943-bd017fde2,
		controller.image.repository=ghcr.io/verrazzano/nginx-ingress-controller,
		defaultBackend.image.tag=0.46.0-20211005200943-bd017fde2,
		defaultBackend.image.repository=ghcr.io/verrazzano/nginx-ingress-default-backend,controller.service.type=LoadBalancer`
	maskedStr = maskSensitiveData(str)
	assert.Equal(t, str, maskedStr)
}

// Run should assert the command parameters are correct then return a success with stdout contents
func (r upgradeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	assert := assert.New(r.t)
	argIndex := 0
	assert.Contains(cmd.Path, "helm", "command should contain helm")
	assert.Contains(cmd.Args[argIndex], "helm", "args should contain helm")
	argIndex++
	assert.Contains(cmd.Args[argIndex], "upgrade", "args should contain upgrade")
	argIndex++
	assert.Contains(cmd.Args[argIndex], release, "args should contain release name")
	argIndex++
	assert.Contains(cmd.Args[argIndex], chartdir, "args should contain chart dir ")
	argIndex++

	for _, arg := range r.expectedArgs {
		assert.Contains(cmd.Args[argIndex], arg)
		argIndex++
	}
	return []byte("success"), []byte(""), nil
}

// Run should assert the command parameters are correct then return a success with stdout contents
func (r getValuesRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	assert := assert.New(r.t)
	assert.Contains(cmd.Path, "helm", "command should contain helm")
	assert.Contains(cmd.Args[0], "helm", "args should contain helm")
	assert.Contains(cmd.Args[1], "get", "args should contain get")
	assert.Contains(cmd.Args[2], "values", "args should contain get")
	assert.Contains(cmd.Args[3], release, "args should contain release name")
	return []byte("success"), []byte(""), nil
}

// Run should return an error with stderr contents
func (r badRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte(""), []byte("error"), errors.New("error")
}

// Run should assert the command parameters are correct then return a success or error depending on release name
func (r foundRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	assert := assert.New(r.t)
	assert.Contains(cmd.Path, "helm", "command should contain helm")
	assert.Contains(cmd.Args[0], "helm", "args should contain helm")
	assert.Contains(cmd.Args[1], "status", "args should contain status")

	if cmd.Args[2] == release {
		return []byte(""), []byte(""), nil
	}
	if cmd.Args[2] == missingRelease {
		return []byte(""), []byte("not found"), errors.New("not found error")
	}
	// simulate a Helm error
	return []byte(""), []byte("error"), errors.New("helm error")
}

// Test_GetReleaseAppVersion tests the GetReleaseAppVersion function
// GIVEN a call to GetReleaseAppVersion
//
//	WHEN varying the inputs and underlying status
//	THEN test the expected result is returned
func Test_GetReleaseAppVersion(t *testing.T) {
	jsonSuccess := []byte(`
[
  {
    "name": "verrazzano",
    "namespace": "verrazzano-system",
    "app_version": "1"
  }
]
`)

	jsonFail := []byte(`
[
  {
    "name": "unknown",
    "namespace": "verrazzano-system",
    "app_version": "1"
  }
]
`)
	type args struct {
		releaseName string
		namespace   string
		jsonOut     []byte
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test GetReleaseAppVersion when app_version exists",
			want: "1",
			args: args{
				releaseName: "verrazzano",
				namespace:   "verrazzano-system",
				jsonOut:     jsonSuccess,
			},
			wantErr: false,
		},
		{
			name: "Test GetReleaseAppVersion when app_version does not exist",
			want: "",
			args: args{
				releaseName: "verrazzano",
				namespace:   "verrazzano-system",
				jsonOut:     jsonFail,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetCmdRunner(genericTestRunner{
				stdOut: tt.args.jsonOut,
				stdErr: []byte{},
				err:    nil,
			})
			defer SetDefaultRunner()
			got, err := GetReleaseAppVersion(tt.args.releaseName, tt.args.namespace)
			if !tt.wantErr {
				assert.NoError(t, err)
				assert.Equalf(t, tt.want, got, "GetReleaseAppVersion(%v, %v)", tt.args.releaseName, tt.args.namespace)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
