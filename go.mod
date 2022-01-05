// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

module github.com/verrazzano/verrazzano

go 1.16

require (
	github.com/Jeffail/gabs/v2 v2.2.0
	github.com/crossplane/crossplane-runtime v0.10.0
	github.com/crossplane/oam-kubernetes-runtime v0.3.2
	github.com/gertd/go-pluralize v0.1.7
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/mock v1.4.4
	github.com/gordonklaus/ineffassign v0.0.0-20210104184537-8eed68eb605f
	github.com/hashicorp/go-retryablehttp v0.6.8
	github.com/jetstack/cert-manager v1.2.0
	github.com/onsi/ginkgo/v2 v2.0.0-rc2
	github.com/onsi/gomega v1.17.0
	github.com/oracle/oci-go-sdk/v53 v53.1.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.16.0
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5
	golang.org/x/tools v0.0.0-20210106214847-113979e3529a
	istio.io/api v0.0.0-20200911191701-0dc35ad5c478
	istio.io/client-go v0.0.0-20200807182027-d287a5abb594
	k8s.io/api v0.21.1
	k8s.io/apiextensions-apiserver v0.19.0
	k8s.io/apimachinery v0.22.3
	k8s.io/cli-runtime v0.22.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.19.2
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/controller-tools v0.4.1
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8
	k8s.io/api => k8s.io/api v0.19.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.2
	k8s.io/client-go => k8s.io/client-go v0.19.2
	k8s.io/code-generator => k8s.io/code-generator v0.19.2
)
