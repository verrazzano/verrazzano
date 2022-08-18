// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bom

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testSubComponent contains the override Key values for a subcomponent.
type testSubComponent struct {
	// kvs is the map of helm Key to expected helm Value.  These values are used in helm overrides
	// for the subcomponent chart
	kvs map[string]string
}

// testSubcomponetHelmKeyValues are the Key:values pairs that will be passed to helm as overrides.
// The map Key is the subcomponent name.
// This list of subcomponents is in the verrazzano-bom.json file and it must stay in sync with that file
// Keep this map in the same order as that JSON for review purposes.
var testSubcomponetHelmKeyValues = map[string]*testSubComponent{
	"verrazzano-platform-operator": {
		kvs: map[string]string{
			"image": "ghcr.io/verrazzano/VERRAZZANO_PLATFORM_OPERATOR_IMAGE:VERRAZZANO_PLATFORM_OPERATOR_TAG",
		},
	},
	"cert-manager": {
		kvs: map[string]string{
			"image.repository": "ghcr.io/verrazzano/cert-manager-controller",
			"image.tag":        "0.13.1-20201016205232-4c8f3fe38",
			"extraArgs[0]=--acme-http01-solver-image": "ghcr.io/verrazzano/cert-manager-acmesolver:0.13.1-20201016205234-4c8f3fe38",
			"webhook.image.repository":                "ghcr.io/verrazzano/cert-manager-webhook",
			"webhook.image.tag":                       "1.2.0-20210602163405-aac6bdf62",
			"cainjector.image.repository":             "ghcr.io/verrazzano/cert-manager-cainjector",
			"cainjector.image.tag":                    "1.2.0-20210602163405-aac6bdf62",
		},
	},
	"ingress-controller": {
		kvs: map[string]string{
			"controller.image.repository":     "ghcr.io/verrazzano/nginx-ingress-controller",
			"controller.image.tag":            "0.46.0-20210510134749-abc2d2088",
			"defaultBackend.image.repository": "ghcr.io/verrazzano/nginx-ingress-default-backend",
			"defaultBackend.image.tag":        "0.46.0-20210510134749-abc2d2088",
		},
	},
	"external-dns": {
		kvs: map[string]string{
			"image.repository": "verrazzano/external-dns",
			"image.registry":   "ghcr.io",
			"image.tag":        "v0.7.1-20201016205338-516bc8b2",
		},
	},
	"istiocoredns": {
		kvs: map[string]string{
			"istiocoredns.coreDNSImage":       "ghcr.io/verrazzano/coredns",
			"istiocoredns.coreDNSTag":         "1.6.2",
			"istiocoredns.coreDNSPluginImage": "ghcr.io/verrazzano/istio-coredns-plugin:0.2-20201016204812-23723dcb",
		},
	},
	"istiod": {
		kvs: map[string]string{
			"pilot.image":        "ghcr.io/verrazzano/pilot:1.7.3",
			"global.proxy.image": "proxyv2",
			"global.tag":         "1.7.3",
		},
	},
	"istio-ingress": {
		kvs: map[string]string{
			"global.proxy.image": "proxyv2",
			"global.tag":         "1.7.3",
		},
	},
	"istio-egress": {
		kvs: map[string]string{
			"global.proxy.image": "proxyv2",
			"global.tag":         "1.7.3",
		},
	},
	"rancher": {
		kvs: map[string]string{
			"rancherImage":    "ghcr.io/verrazzano/rancher",
			"rancherImageTag": "v2.5.7-20210407205410-1c7b39d0c",
		},
	},
	// NOTE additional-rancher images are not used by the local rancher helm chart used by verrazzano
	// so ignore those entries

	"verrazzano": {
		kvs: map[string]string{
			"monitoringOperator.imageName":       "ghcr.io/verrazzano/verrazzano-monitoring-operator",
			"monitoringOperator.imageVersion":    "0.15.0-20210521020822-9b87485",
			"monitoringOperator.istioProxyImage": "ghcr.io/verrazzano/proxyv2:1.7.3",
			"monitoringOperator.grafanaImage":    "ghcr.io/verrazzano/grafana:v6.4.4",
			"monitoringOperator.prometheusImage": "ghcr.io/verrazzano/prometheus:v2.13.1",
			"monitoringOperator.esImage":         "ghcr.io/verrazzano/elasticsearch:7.6.1-20201130145440-5c76ab1",
			"monitoringOperator.esWaitImage":     "ghcr.io/verrazzano/verrazzano-monitoring-instance-eswait:0.15.0-20210521020822-9b87485",
			"monitoringOperator.kibanaImage":     "ghcr.io/verrazzano/kibana:7.6.1-20201130145840-7717e73",
			"monitoringOperator.oidcProxyImage":  "ghcr.io/verrazzano/nginx-ingress-controller:0.46.0-20210510134749-abc2d2088",
			"logging.fluentdImage":               "ghcr.io/verrazzano/fluentd-kubernetes-daemonset:v1.12.3-20210517195222-f345ec2",
			"console.imageName":                  "ghcr.io/verrazzano/console",
			"console.imageVersion":               "0.15.0-20210512140333-bbb6bd7",
			"api.imageName":                      "ghcr.io/verrazzano/nginx-ingress-controller",
			"api.imageVersion":                   "0.46.0-20210510134749-abc2d2088",
		},
	},
	"monitoring-init-images": {
		kvs: map[string]string{
			"monitoringOperator.prometheusInitImage": "ghcr.io/oracle/oraclelinux:7-slim",
			"monitoringOperator.esInitImage":         "ghcr.io/oracle/oraclelinux:7.8",
		},
	},
	"oam-kubernetes-runtime": {
		kvs: map[string]string{
			"image.repository": "ghcr.io/verrazzano/oam-kubernetes-runtime",
			"image.tag":        "v0.3.0-20210222205541-9e8d4fb",
		},
	},
	"verrazzano-application-operator": {
		kvs: map[string]string{
			"image":        "ghcr.io/verrazzano/VERRAZZANO_APPLICATION_OPERATOR_IMAGE:VERRAZZANO_APPLICATION_OPERATOR_TAG",
			"fluentdImage": "ghcr.io/verrazzano/fluentd-kubernetes-daemonset:v1.12.3-20210517195222-f345ec2",
		},
	},
	"weblogic-operator": {
		kvs: map[string]string{
			"image": "ghcr.io/oracle/weblogic-kubernetes-operator:3.2.2",
		},
	},
	"coherence-operator": {
		kvs: map[string]string{
			"image": "ghcr.io/oracle/coherence-operator:3.1.3",
		},
	},
	"mysql": {
		kvs: map[string]string{
			"image":    "ghcr.io/verrazzano/mysql",
			"imageTag": "8.0.20",
		},
	},
	"oraclelinux": {
		kvs: map[string]string{
			"busybox.image": "ghcr.io/oracle/oraclelinux",
			"busybox.tag":   "7-slim",
		},
	},
	"keycloak": {
		kvs: map[string]string{
			"keycloak.image.repository": "ghcr.io/verrazzano/keycloak",
			"keycloak.image.tag":        "10.0.1-20201016212759-30d98b0",
		},
	},
	//"keycloak-oracle-theme": {
	//	kvs: map[string]string{
	//		"image": "ghcr.io/verrazzano/keycloak-oracle-theme:0.15.0-20210510085250-01638c7",
	//	},
	//},
}

// This is the real BOM file path needed for unit tests
const realBomFilePath = "testdata/verrazzano-bom.json"
const testBomFilePath = "testdata/test_bom.json"
const testBomSubcomponentOverridesPath = "testdata/test_bom_sc_overrides.json"
const testBomImageOverridesPath = "testdata/test_bom_image_overrides.json"

// TestFakeBom tests loading a fake bom json into a struct
// GIVEN a json file
// WHEN I call loadBom
// THEN the correct Verrazzano bom is returned
func TestFakeBom(t *testing.T) {
	assert := assert.New(t)
	bom, err := NewBom(testBomFilePath)
	assert.NoError(err, "error calling NewBom")
	assert.Equal("ghcr.io", bom.bomDoc.Registry, "Wrong registry name")
	assert.Len(bom.bomDoc.Components, 14, "incorrect number of Bom components")

	validateImages(assert, &bom, true)
}

// TestRealBom tests loading the real bom json into a struct
// GIVEN a json file
// WHEN I call loadBom
// THEN the correct Verrazzano bom is returned
func TestRealBom(t *testing.T) {
	assert := assert.New(t)
	bom, err := NewBom(realBomFilePath)
	assert.NoError(err, "error calling NewBom")
	assert.Equal("ghcr.io", bom.bomDoc.Registry, "Wrong registry name")
	assert.Len(bom.bomDoc.Components, 14, "incorrect number of Bom components")

	// Ignore the values in the real bom file since some will change every build
	validateImages(assert, &bom, false)
}

// validateImages validates the images in the subcomponents.
// Optionall check the image Value.
func validateImages(assert *assert.Assertions, bom *Bom, checkImageVal bool) {
	// Validate each component
	for _, comp := range bom.bomDoc.Components {
		for _, sub := range comp.SubComponents {
			// Get the expected Key:Value pair overrides for this subcomponent
			expectedSub := testSubcomponetHelmKeyValues[sub.Name]
			if expectedSub == nil {
				fmt.Println("Skipping subcomponent " + sub.Name)
				continue
			}

			// Get the actual Key Value override list for this subcomponent
			foundKvs, err := bom.BuildImageOverrides(sub.Name)
			assert.NoError(err, "error calling BuildImageOverrides")
			assert.Equal(len(expectedSub.kvs), len(foundKvs), "Incorrect override list len for "+sub.Name)

			// Loop through the found kv pairs and make sure the actual kvs match the expected
			for _, kv := range foundKvs {
				expectedVal, ok := expectedSub.kvs[kv.Key]
				assert.True(ok, "Found unexpected Key in override list for "+sub.Name)
				if checkImageVal {
					assert.Equal(expectedVal, kv.Value, "Found unexpected Value in override list for "+sub.Name)
				}
			}
		}
	}
}

// TestBomSubcomponentOverrides the ability to override registry and repo settings at the subcomponent level
// GIVEN a json file where a subcomponent overrides the registry and repository location of its images
// WHEN I load it and check those settings
// THEN the correct overrides are present without affecting the global registry setting
func TestBomSubcomponentOverrides(t *testing.T) {
	assert := assert.New(t)
	bom, err := NewBom(testBomSubcomponentOverridesPath)
	assert.Equal("ghcr.io", bom.GetRegistry(), "Global registry not correct")
	assert.NoError(err)

	nginxSubcomponent, err := bom.GetSubcomponent("ingress-controller")
	assert.NotNil(t, nginxSubcomponent)
	assert.NoError(err)

	assert.Equal("ghcr.io", bom.GetRegistry(), "Global registry not correct")
	assert.Equal("myreg.io", bom.ResolveRegistry(nginxSubcomponent, BomImage{}), "NGINX subcomponent registry not correct")
	assert.Equal("myrepoprefix/testnginx", bom.ResolveRepo(nginxSubcomponent, BomImage{}), "NGINX subcomponent repo not correct")

	vpoSubcomponent, err := bom.GetSubcomponent("verrazzano-platform-operator")
	assert.NotNil(t, vpoSubcomponent)
	assert.NoError(err)

	assert.Equal("ghcr.io", bom.ResolveRegistry(vpoSubcomponent, BomImage{}), "VPO subcomponent registry not correct")
	assert.Equal("verrazzano", bom.ResolveRepo(vpoSubcomponent, BomImage{}), "VPO subcomponent repo not correct")
}

func TestBomImageOverrides(t *testing.T) {
	bom, err := NewBom(testBomImageOverridesPath)
	assert.NoError(t, err)
	assert.Equal(t, "ghcr.io", bom.GetRegistry())
	sc, err := bom.GetSubcomponent("verrazzano-platform-operator")
	assert.NoError(t, err)
	img := sc.Images[0]
	assert.Equal(t, "testRegistry", bom.ResolveRegistry(sc, img))
	assert.Equal(t, "testRepository", bom.ResolveRepo(sc, img))
}

// TestBomComponentVersion tests the ability to fetch component version
func TestBomComponentVersion(t *testing.T){
	bom, err := NewBom(realBomFilePath)
	assert.NoError(t, err)
	c, err := bom.GetComponent("verrazzano")
	assert.NoError(t, err)
	assert.NotNil(t, c)
	assert.NotNil(t, c.Version)
}