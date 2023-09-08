// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"github.com/oracle/oci-go-sdk/v53/core"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	ocifake "github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci/fake"
	vzerror "github.com/verrazzano/verrazzano/cluster-operator/internal/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

func testValidationContextWithOCIClient(cli clipkg.Client, ociClient oci.Client) *validationContext {
	return &validationContext{
		Ctx: context.TODO(),
		Cli: cli,
		OCIClientGetter: func(creds *oci.Credentials) (oci.Client, error) {
			return ociClient, nil
		},
		CredentialsLoader: &ocifake.CredentialsLoaderImpl{
			Credentials: &oci.Credentials{},
		},
		Errors: &vzerror.ErrorAggregator{},
	}
}

func testValidationContext(cli clipkg.Client) *validationContext {
	return testValidationContextWithOCIClient(cli, nil)
}

func testOCNEConfigMap() (*corev1.ConfigMap, error) {
	d, err := os.ReadFile("../../../controllers/quickcreate/controller/ocne/testdata/ocne-versions.yaml")
	if err != nil {
		return nil, err
	}
	cm := &corev1.ConfigMap{}
	if err := yaml.Unmarshal(d, cm); err != nil {
		return nil, err
	}
	return cm, nil
}

func TestAddOCINodeErrors(t *testing.T) {
	var (
		flexShape    = "VM.Standard.E4.Flex"
		nonFlexShape = "BM.Standard.E5"
		one          = 1
		fifty        = 50
	)
	var tests = []struct {
		name     string
		node     OCINode
		hasError bool
	}{
		{
			"no error when ocpus/memory are not provided for a flex shape",
			OCINode{
				Shape:         &flexShape,
				OCPUs:         &one,
				MemoryGbs:     &one,
				BootVolumeGbs: &fifty,
				Replicas:      &one,
			},
			false,
		},
		{
			"no error when ocpus/memory are provided for a flex shape",
			OCINode{
				Shape:         &flexShape,
				OCPUs:         &one,
				MemoryGbs:     &one,
				BootVolumeGbs: &fifty,
				Replicas:      &one,
			},
			false,
		},
		{
			"error when providing ocpus without flex shape",
			OCINode{
				Shape:         &nonFlexShape,
				OCPUs:         &one,
				BootVolumeGbs: &fifty,
				Replicas:      &one,
			},
			true,
		},
		{
			"error when providing memory without flex shape",
			OCINode{
				Shape:         &nonFlexShape,
				MemoryGbs:     &one,
				BootVolumeGbs: &fifty,
				Replicas:      &one,
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testValidationContext(nil)
			addOCINodeErrors(ctx, tt.node, "")
			assert.Equal(t, tt.hasError, ctx.Errors.HasError())
		})
	}
}

func TestAddOCINetworkErrors(t *testing.T) {
	var (
		vcnID    = "my-vcn"
		subnetID = "my-subnet"
		lbSubnet = Subnet{
			Role: SubnetRoleServiceLB,
			ID:   subnetID,
		}
		workerSubnet = Subnet{
			Role: SubnetRoleWorker,
			ID:   subnetID,
		}
		cpSubnet = Subnet{
			Role: SubnetRoleControlPlane,
			ID:   subnetID,
		}
		cpeSubnet = Subnet{
			Role: SubnetRoleControlPlaneEndpoint,
			ID:   subnetID,
		}
		subnets = []Subnet{
			workerSubnet,
			lbSubnet,
			cpSubnet,
			cpeSubnet,
		}
	)
	ociClient := &ocifake.ClientImpl{
		VCN: &core.Vcn{
			Id: &vcnID,
		},
	}
	var tests = []struct {
		name     string
		network  *Network
		hasError bool
	}{
		{
			"no error when using creating a VCN without existing VCN/subnet ids",
			&Network{
				CreateVCN: true,
			},
			false,
		},
		{
			"error when creating a VCN, but a VCN id is supplied",
			&Network{
				CreateVCN: true,
				VCN:       vcnID,
			},
			true,
		},
		{
			"error when creating a VCN, but subnets are supplied",
			&Network{
				CreateVCN: true,
				Subnets:   subnets,
			},
			true,
		},
		{
			"error when using an existing a VCN, and not enough subnets are supplied",
			&Network{
				CreateVCN: false,
				VCN:       vcnID,
				Subnets:   []Subnet{workerSubnet},
			},
			true,
		},
		{
			"no error when using an existing vcn and subnets",
			&Network{
				CreateVCN: false,
				VCN:       vcnID,
				Subnets:   subnets,
			},
			false,
		},
		{
			"error when vcn is not found",
			&Network{
				CreateVCN: false,
				VCN:       "unknown",
				Subnets:   subnets,
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testValidationContext(nil)
			addOCINetworkErrors(ctx, ociClient, tt.network, "")
			assert.Equal(t, tt.hasError, ctx.Errors.HasError())
		})
	}
}

func TestAddOCNEErrors(t *testing.T) {
	ocneConfigMap, err := testOCNEConfigMap()
	assert.NoError(t, err)
	ocneVersion := "1.7"
	cliWithCM := fake.NewClientBuilder().WithObjects(ocneConfigMap).WithScheme(scheme.Scheme).Build()
	var tests = []struct {
		name        string
		ocneVersion string

		cli      clipkg.Client
		hasError bool
	}{
		{"no errors when using valid OCNE version and configmap is present",
			ocneVersion,

			cliWithCM,
			false,
		},
		{"error when using invalid OCNE version",
			"boom",

			cliWithCM,
			true,
		},
		{"error when OCNE Versions are not presnet",
			ocneVersion,

			fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testValidationContext(tt.cli)
			addOCNEErrors(ctx, OCNE{
				Version: tt.ocneVersion,
			}, "")
			assert.Equal(t, tt.hasError, ctx.Errors.HasError())
		})
	}
}

func TestAddProxyErrors(t *testing.T) {
	var tests = []struct {
		name     string
		proxy    *Proxy
		hasError bool
	}{
		{
			"no error when proxy is nil",
			nil,
			false,
		},
		{
			"no error when proxy has valid urls",
			&Proxy{
				HTTPProxy:  "http://foo.com",
				HTTPSProxy: "https://foo.com",
				NoProxy:    "",
			},
			false,
		},
		{
			"error when proxy has invalid urls",
			&Proxy{
				HTTPProxy: "x",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testValidationContext(nil)
			addProxyErrors(ctx, tt.proxy, "")
			assert.Equal(t, tt.hasError, ctx.Errors.HasError())
		})
	}
}

func TestAddPrivateRegistryErrors(t *testing.T) {
	var tests = []struct {
		name            string
		privateRegistry *PrivateRegistry
		hasError        bool
	}{
		{
			"no error when private registry is nil",
			nil,
			false,
		},
		{
			"no error when private registry has valid urls",
			&PrivateRegistry{
				URL: "http://my-registry.com:80",
			},
			false,
		},
		{
			"error when private registry has invalid urls",
			&PrivateRegistry{
				URL: "x",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testValidationContext(nil)
			addPrivateRegistryErrors(ctx, tt.privateRegistry, "")
			assert.Equal(t, tt.hasError, ctx.Errors.HasError())
		})
	}
}
