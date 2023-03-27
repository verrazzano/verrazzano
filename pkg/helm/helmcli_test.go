// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
)

const ns = "my-namespace"
const chartdir = "my_charts"
const helmRelease = "my-release"
const missingRelease = "no-release"

func testActionConfigWithRelease(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return CreateActionConfig(true, helmRelease, release.StatusDeployed, log, createRelease)
}

func testActionConfigWithFailedRelease(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return CreateActionConfig(true, helmRelease, release.StatusFailed, log, createRelease)
}

func testActionConfigWithPendingRelease(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return CreateActionConfig(true, helmRelease, release.StatusPendingInstall, log, createRelease)
}

func testActionConfig(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return CreateActionConfig(false, helmRelease, release.StatusFailed, log, createRelease)
}

func getChart() *chart.Chart {
	return &chart.Chart{
		Metadata: &chart.Metadata{
			APIVersion: "v1",
			Name:       "hello",
			Version:    "0.1.0",
			AppVersion: "1.0",
		},
		Templates: []*chart.File{
			{Name: "templates/hello", Data: []byte("hello: world")},
		},
	}
}

func createRelease(name string, status release.Status) *release.Release {
	now := time.Now()
	return &release.Release{
		Name:      name,
		Namespace: ns,
		Info: &release.Info{
			FirstDeployed: now,
			LastDeployed:  now,
			Status:        status,
			Description:   "Named Release Stub",
		},
		Chart: getChart(),
		Config: map[string]interface{}{
			"name1": "value1",
			"name2": "value2",
			"nestedKey": map[string]interface{}{
				"simpleKey": "simpleValue",
				"anotherNestedKey": map[string]interface{}{
					"yetAnotherNestedKey": map[string]interface{}{
						"youReadyForAnotherNestedKey": "No",
					},
				},
			},
		},
		Version: 1,
	}
}

// TestGetValues tests the Helm get values command
// GIVEN a set of upgrade parameters
//
//	WHEN I call Upgrade
//	THEN the Helm upgrade returns success and the cmd object has correct values
func TestGetValues(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	vals, err := GetValues(vzlog.DefaultLogger(), helmRelease, ns)
	assertion.NoError(err, "GetValues returned an error")
	assertion.NotZero(vals, "GetValues stdout should not be empty")
}

// TestGetValuesMap tests the Helm get values command
// GIVEN a set of upgrade parameters
//
//	WHEN I call Upgrade
//	THEN the Helm upgrade returns success and the cmd object has correct values
func TestGetValuesMap(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	vals, err := GetValuesMap(vzlog.DefaultLogger(), helmRelease, ns)
	assertion.NoError(err, "GetValues returned an error")
	assertion.Len(vals, 3)
}

// TestUpgrade tests the Helm upgrade command
// GIVEN a set of upgrade parameters
//
//	WHEN I call Upgrade
//	THEN the Helm upgrade returns success and the cmd object has correct values
func TestUpgrade(t *testing.T) {
	var overrides []HelmOverrides
	overrides = append(overrides, HelmOverrides{SetOverrides: "name1=modifiedValue1"})
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()
	SetLoadChartFunction(func(chartDir string) (*chart.Chart, error) {
		return getChart(), nil
	})
	defer SetDefaultLoadChartFunction()

	err := Upgrade(vzlog.DefaultLogger(), helmRelease, ns, chartdir, false, false, overrides)
	assertion.NoError(err, "Upgrade returned an error")
}

// TestUpgradeFail tests the Helm upgrade command failure condition
// GIVEN a set of upgrade parameters and a fake runner that fails
//
//	WHEN I call Upgrade
//	THEN the Helm upgrade returns an error
func TestUpgradeFail(t *testing.T) {
	var overrides []HelmOverrides
	overrides = append(overrides, HelmOverrides{SetOverrides: "name1=modifiedValue1"})
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()
	// no chart load function should generate an error

	err := Upgrade(vzlog.DefaultLogger(), helmRelease, ns, "", false, false, overrides)
	assertion.Error(err, "Upgrade should have returned an error")
}

// TestUninstall tests the Helm Uninstall fn
// GIVEN a call to Uninstall
//
//	WHEN the command executes successfully
//	THEN the function returns no error
func TestUninstall(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	err := Uninstall(vzlog.DefaultLogger(), helmRelease, ns, false)
	assertion.NoError(err)
}

// TestUninstallError tests the Helm Uninstall fn
// GIVEN a call to Uninstall
//
//	WHEN the command executes and returns an error
//	THEN the function returns an error
func TestUninstallError(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfig)
	defer SetDefaultActionConfigFunction()

	err := Uninstall(vzlog.DefaultLogger(), helmRelease, ns, false)
	assertion.Error(err)
}

// TestIsReleaseInstalled tests checking if a Helm helmRelease is installed
// GIVEN a helmRelease name and namespace
//
//	WHEN I call IsReleaseInstalled
//	THEN the function returns success and found equal true
func TestIsReleaseInstalled(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	found, err := IsReleaseInstalled(helmRelease, ns)
	assertion.NoError(err, "IsReleaseInstalled returned an error")
	assertion.True(found, "Release not found")
}

// TestIsReleaseNotInstalled tests checking if a Helm helmRelease is not installed
// GIVEN a helmRelease name and namespace
//
//	WHEN I call IsReleaseInstalled
//	THEN the function returns success and the correct found status
func TestIsReleaseNotInstalled(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	found, err := IsReleaseInstalled(missingRelease, ns)
	assertion.NoError(err, "IsReleaseInstalled returned an error")
	assertion.False(found, "Release should not be found")
}

// TestIsReleaseInstalledFailed tests failure when checking if a Helm helmRelease is installed
// GIVEN a bad helmRelease name and namespace
//
//	WHEN I call IsReleaseInstalled
//	THEN the function returns a failure
func TestIsReleaseInstalledFailed(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	found, err := IsReleaseInstalled("", ns)
	assertion.Error(err, "IsReleaseInstalled should have returned an error")
	assertion.False(found, "Release should not be found")
}

// TestIsReleaseDeployed tests checking if a Helm helmRelease is installed
// GIVEN a helmRelease name and namespace
//
//	WHEN I call IsReleaseDeployed
//	THEN the function returns success and found equal true
func TestIsReleaseDeployed(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	found, err := IsReleaseDeployed(helmRelease, ns)
	assertion.NoError(err, "IsReleaseDeployed returned an error")
	assertion.True(found, "Release not found")
}

// TestIsReleaseNotDeployed tests checking if a Helm helmRelease is not installed
// GIVEN a helmRelease name and namespace
//
//	WHEN I call IsReleaseDeployed
//	THEN the function returns success and the correct found status
func TestIsReleaseNotDeployed(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	found, err := IsReleaseDeployed(missingRelease, ns)
	assertion.NoError(err, "IsReleaseDeployed returned an error")
	assertion.False(found, "Release should not be found")
}

// TestIsReleaseFailedChartNotFound tests checking if a Helm helmRelease is in a failed state
// GIVEN a helmRelease name and namespace
//
//	WHEN I call IsReleaseFailed and the status is ChartNotFound
//	THEN the function returns false and no error
func TestIsReleaseFailedChartNotFound(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	failed, err := IsReleaseFailed("foo", "bar")
	assertion.NoError(err, "IsReleaseFailed returned an error")
	assertion.False(failed, "ReleaseFailed should be false")
}

// TestIsReleaseFailedChartDeployed tests checking if a Helm helmRelease is in a failed state
// GIVEN a helmRelease name and namespace
//
//	WHEN I call IsReleaseFailed and the status is deployed
//	THEN the function returns false and no error
func TestIsReleaseFailedChartDeployed(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithFailedRelease)
	defer SetDefaultActionConfigFunction()

	failed, err := IsReleaseFailed(helmRelease, ns)
	assertion.NoError(err, "IsReleaseFailed returned an error")
	assertion.True(failed, "ReleaseFailed should be true")
}

// Test_getReleaseStateDeployed tests the getReleaseState fn
// GIVEN a call to getReleaseState
//
//	WHEN the chart state is deployed
//	THEN the function returns ChartStatusDeployed and no error
func Test_getReleaseStateDeployed(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	state, err := getReleaseState(helmRelease, ns)
	assertion.NoError(err)
	assertion.Equalf(ChartStatusDeployed, state, "unpexected state: %s", state)
}

// Test_getReleaseStateDeployed tests the getReleaseState fn
// GIVEN a call to getReleaseState
//
//	WHEN the chart state is pending-install
//	THEN the function returns ChartStatusPendingInstall and no error
func Test_getReleaseStatePendingInstall(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithPendingRelease)
	defer SetDefaultActionConfigFunction()

	state, err := getReleaseState(helmRelease, ns)
	assertion.NoError(err)
	assertion.Equalf(ChartStatusPendingInstall, state, "unpexected state: %s", state)
}

// Test_getReleaseStateChartNotFound tests the getReleaseState fn
// GIVEN a call to getReleaseState
//
//	WHEN the chart/helmRelease can not be found
//	THEN the function returns "" and no error
func Test_getReleaseStateChartNotFound(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	state, err := getReleaseState("weblogic-operator", "verrazzano-system")
	assertion.NoError(err)
	assertion.Equalf("", state, "unpexected state: %s", state)
}

// Test_getChartStatusDeployed tests the getChartStatus fn
// GIVEN a call to getChartStatus
//
//	WHEN Helm returns a deployed state
//	THEN the function returns "deployed" and no error
func Test_getChartStatusDeployed(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	state, err := getChartStatus(helmRelease, ns)
	assertion.NoError(err)
	assertion.Equalf(ChartStatusDeployed, state, "unpexected state: %s", state)
}

// Test_getChartStatusChartNotFound tests the getChartStatus fn
// GIVEN a call to getChartStatus
//
//	WHEN the Chart is not found
//	THEN the function returns chart not found and no error
func Test_getChartStatusChartNotFound(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	state, err := getChartStatus("weblogic-operator", "verrazzano-system")
	assertion.NoError(err)
	assertion.Equalf(ChartNotFound, state, "unpexected state: %s", state)
}

// TestGetReleaseValue tests the GetReleaseValues fn
// GIVEN a call to GetReleaseValues
//
//	WHEN a valid helm helmRelease and namespace are deployed
//	THEN the function returns the value/true/nil if the helm key exists, or ""/false/nil if it doesn't
func TestGetReleaseValues(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	keys := []string{"name1", "name2", "simpleKey", "foo"}
	value, err := GetReleaseValues(vzlog.DefaultLogger(), keys, helmRelease, ns)

	expectedMap := map[string]interface{}{
		"name1": "value1",
		"name2": "value2",
	}
	assertion.NoError(err)
	assertion.Equal(expectedMap, value, "Map did not contain expected values")
}

// TestGetReleaseStringValue tests the GetReleaseStringValues fn
// GIVEN a call to GetReleaseStringValues
//
//	WHEN a valid helm helmRelease and namespace are deployed
//	THEN the function returns the value/true/nil if the helm key exists, or ""/false/nil if it doesn't
func TestGetReleaseStringValue(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	keys := []string{"nestedKey", "foo"}
	value, err := GetReleaseStringValues(vzlog.DefaultLogger(), keys, helmRelease, ns)

	expectedMap := map[string]string{
		"nestedKey": "map[anotherNestedKey:map[yetAnotherNestedKey:map[youReadyForAnotherNestedKey:No]] simpleKey:simpleValue]",
	}
	assertion.NoError(err)
	assertion.Equal(expectedMap, value, "Map did not contain expected values")
}

// TestGetReleaseValueReleaseNotFound tests the GetReleaseValues fn
// GIVEN a call to GetReleaseValues
//
//	WHEN a the helm helmRelease is NOT deployed
//	THEN the function returns the value/true/nil if the helm key exists, or ""/false/nil if it doesn't
func TestGetReleaseValueReleaseNotFound(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	keys := []string{"txtOwnerId", "external-dns"}
	values, err := GetReleaseValues(vzlog.DefaultLogger(), keys, "external-dns", "cert-manager")
	assertion.NoErrorf(err, "Unexpected error: %v", err)
	assertion.Equal(map[string]interface{}{}, values, "Found unexpected helmRelease value")
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

// Test_GetReleaseAppVersion tests the GetReleaseAppVersion function
// GIVEN a call to GetReleaseAppVersion
//
//	WHEN varying the inputs and underlying status
//	THEN test the expected result is returned
func Test_GetReleaseAppVersion(t *testing.T) {
	assertion := assert.New(t)
	SetActionConfigFunction(testActionConfigWithRelease)
	defer SetDefaultActionConfigFunction()

	got, err := GetReleaseAppVersion(helmRelease, ns)
	assertion.NoError(err)
	assertion.Equal("1.0", got)
}
