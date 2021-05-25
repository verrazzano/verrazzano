// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

// testSubComponent contains the override key values for a subcomponent.
type testSubComponent struct {
	// kvs is the map of helm key to expected helm value.  These values are used in helm overrides
	// for the subcomponent chart
	kvs map[string]string
}

// testSubcomponetHelmKeyValues are the key/values pairs that will be passed to helm as overrides.
// The map key is the subcomponent name.
// This definitive list of subcomponents is in the verrazzano-bom.json file.  Keep this map in the
// same order as that JSON for review purposes.
var testSubcomponetHelmKeyValues = map[string]*testSubComponent{
	"verrazzano-platform-operator": {
		kvs: map[string]string{
			"image": "ghcr.io/verrazzano/verrazzano-platform-operator:0.15.0-20210519205437-9cd1da0b",
		},
	},
	"cert-manager": {
		kvs: map[string]string{
			"image.repository": "ghcr.io/verrazzano/cert-manager-controller",
			"image.tag": "0.13.1-20201016205232-4c8f3fe38",
		},
	},
	"external-dns": {
		kvs: map[string]string{
			"image.repository": "verrazzano/external-dns",
			"image.registry": "ghcr.io",
			"image.tag": "v0.7.1-20201016205338-516bc8b2",
		},
	},
	"istiocoredns": {
		kvs: map[string]string{
			"istiocoredns.coreDNSImage": "ghcr.io/verrazzano/coredns",
			"istiocoredns.coreDNSTag": "1.6.2",
			"istiocoredns.coreDNSPluginImage": "ghcr.io/verrazzano/istio-coredns-plugin:0.2-20201016204812-23723dcb",
		},
	},
	"istiod": {
		kvs: map[string]string{
			"pilot.image": "ghcr.io/verrazzano/pilot:1.7.3",
			"global.proxy.image": "ghcr.io/verrazzano/proxyv2",
			"global.tag": "1.7.3",
		},
	},
	"istio-ingress": {
		kvs: map[string]string{
			"global.proxy.image": "ghcr.io/verrazzano/proxyv2",
			"global.tag": "1.7.3",
		},
	},
	"istio-egress": {
		kvs: map[string]string{
			"global.proxy.image": "ghcr.io/verrazzano/proxyv2",
			"global.tag": "1.7.3",
		},
	},
	"rancher": {
		kvs: map[string]string{
			"rancherImage": "ghcr.io/verrazzano/rancher",
			"rancherImageTag": "v2.5.7-20210407205410-1c7b39d0c",
			"image": "ghcr.io/verrazzano/rancher-agent:v2.5.7-20210407205410-1c7b39d0c",
		},
	},
	// NOTE additional-rancher images are not used by the local rancher helm chart used by verrazzano
	// so ignore those entries

	"verrazzano": {
		kvs: map[string]string{
			"verrazzanoOperator.imageName": "ghcr.io/verrazzano/verrazzano-operator",
			"verrazzanoOperator.imageVersion": "0.15.0-20210512213227-2785c3a",
			"verrazzanoOperator.nodeExporterImage": "ghcr.io/verrazzano/node-exporter:1.0.0-20210513143333-a470f06",
			"monitoringOperator.imageName": "ghcr.io/verrazzano/verrazzano-monitoring-operator",
			"monitoringOperator.imageVersion": "0.15.0-20210521020822-9b87485",
			"monitoringOperator.istioProxyImage": "ghcr.io/verrazzano/proxyv2:1.7.3",
			"monitoringOperator.grafanaImage": "ghcr.io/verrazzano/grafana:v6.4.4",
			"monitoringOperator.prometheusImage": "ghcr.io/verrazzano/prometheus:v2.13.1",
			"monitoringOperator.esImage": "ghcr.io/verrazzano/elasticsearch:7.6.1-20201130145440-5c76ab1",
			"monitoringOperator.esWaitImage": "ghcr.io/verrazzano/verrazzano-monitoring-instance-eswait:0.15.0-20210521020822-9b87485",
			"monitoringOperator.kibanaImage": "ghcr.io/verrazzano/kibana:7.6.1-20201130145840-7717e73",
			"monitoringOperator.configReloaderImage": "ghcr.io/verrazzano/configmap-reload:0.3-20201016205243-4f24a0e",
			"monitoringOperator.oidcProxyImage": "ghcr.io/verrazzano/nginx-ingress-controller:0.46.0-20210510134749-abc2d2088",
			"logging.fluentdImage": "ghcr.io/verrazzano/fluentd-kubernetes-daemonset:v1.12.3-20210517195222-f345ec2",
			"console.imageName": "ghcr.io/verrazzano/console",
			"console.imageVersion": "0.15.0-20210512140333-bbb6bd7",
			"api.imageName": "ghcr.io/verrazzano/nginx-ingress-controller",
			"api.imageVersion": "0.46.0-20210510134749-abc2d2088",
		},
	},
}

// TestFakeBom tests loading a fake bom json into a struct
// GIVEN a json file
// WHEN I call loadBom
// THEN the correct verrazzano bom is returned
func TestFakeBom(t *testing.T) {
	assert := assert.New(t)
	bom, err := NewBom("testdata/test_bom.json")
	assert.NoError(err, "error calling NewBom")
	assert.Equal("ghcr.io", bom.bomDoc.Registry, "Wrong registry name")
	assert.Len(bom.bomDoc.Components,14, "incorrect number of Bom components")

	// Validate each component
	for _, comp := range bom.bomDoc.Components {
		for _, sub := range comp.SubComponents {
			// Get the expected key/value pair overrides
			expectedSub := testSubcomponetHelmKeyValues[sub.Name]
			if expectedSub == nil{
				fmt.Println("Skipping subcomponent " + sub.Name)
				continue
			}
			if sub.Name == "rancher" {
				fmt.Println("debug")
			}

			// Get the key value override list for this subcomponent
			foundKvs, err := bom.buildOverrides(sub.Name)
			assert.NoError(err, "error calling buildOverrides")
			assert.Equal(len(expectedSub.kvs), len(foundKvs), "Incorrect override list len for "  + sub.Name)

			// Loop through the found kv pairs and make sure they match
			for _, kv := range foundKvs {
				expectedVal, ok := expectedSub.kvs[kv.key]
				assert.True(ok,"Found unexpected key in override list for " + sub.Name)
				assert.Equal(expectedVal, kv.value, "Found unexpected key value in override list for " + sub.Name)
				}
			}
	}



}
