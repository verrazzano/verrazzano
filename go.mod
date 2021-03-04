// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

module github.com/verrazzano/verrazzano

go 1.15

require (
	github.com/Jeffail/gabs/v2 v2.2.0
	github.com/crossplane/crossplane-runtime v0.10.0
	github.com/crossplane/oam-kubernetes-runtime v0.3.2
	github.com/gertd/go-pluralize v0.1.7
	github.com/go-logr/logr v0.1.0
	github.com/golang/mock v1.4.4
	github.com/gordonklaus/ineffassign v0.0.0-20210104184537-8eed68eb605f
	github.com/hashicorp/go-retryablehttp v0.6.8
	github.com/jetstack/cert-manager v0.13.1
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/stretchr/testify v1.6.1
	github.com/verrazzano/verrazzano-crd-generator v0.3.34
	github.com/verrazzano/verrazzano-monitoring-operator v0.0.25
	go.uber.org/zap v1.16.0
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5
	istio.io/api v0.0.0-20200911191701-0dc35ad5c478
	istio.io/client-go v0.0.0-20200807182027-d287a5abb594
	k8s.io/api v0.19.0
	k8s.io/apiextensions-apiserver v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.18.6
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/controller-tools v0.4.1
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.3.1
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8
	k8s.io/api => k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.6
	k8s.io/client-go => k8s.io/client-go v0.18.6
	k8s.io/code-generator => k8s.io/code-generator v0.18.6
)
