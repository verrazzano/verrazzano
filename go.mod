// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

module github.com/verrazzano/verrazzano

go 1.16

require (
	github.com/Jeffail/gabs/v2 v2.6.1
	github.com/crossplane/crossplane-runtime v0.13.0
	github.com/gertd/go-pluralize v0.1.7
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0 // indirect
	github.com/golang/mock v1.5.0
	github.com/gordonklaus/ineffassign v0.0.0-20210914165742-4cc7213b9bc8
	github.com/hashicorp/go-retryablehttp v0.7.0
	github.com/jetstack/cert-manager v1.5.4
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.14.0
	github.com/prometheus/client_golang v1.11.0 // indirect
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/verrazzano/oam-kubernetes-runtime v0.3.3-0.20211022163517-5d196f8b31e8
	go.uber.org/zap v1.17.0
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	golang.org/x/tools v0.1.5
	istio.io/api v0.0.0-20211020081732-2de5b65af1fe
	istio.io/client-go v1.11.4
	k8s.io/api v0.22.2
	k8s.io/apiextensions-apiserver v0.21.3
	k8s.io/apimachinery v0.22.2
	k8s.io/cli-runtime v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/code-generator v0.22.2
	sigs.k8s.io/controller-runtime v0.9.2
	sigs.k8s.io/controller-tools v0.6.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Jeffail/gabs/v2 => github.com/Jeffail/gabs/v2 v2.2.0
	github.com/golang/mock => github.com/golang/mock v1.4.4
	github.com/gordonklaus/ineffassign => github.com/gordonklaus/ineffassign v0.0.0-20210104184537-8eed68eb605f
	github.com/hashicorp/go-retryablehttp => github.com/hashicorp/go-retryablehttp v0.6.8
	github.com/jetstack/cert-manager => github.com/jetstack/cert-manager v1.2.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.13.0
	github.com/spf13/cobra => github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify => github.com/stretchr/testify v1.6.1
	go.uber.org/zap => go.uber.org/zap v1.16.0
	golang.org/x/lint => golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5
	golang.org/x/tools => golang.org/x/tools v0.0.0-20201224043029-2b0845dc783e
	istio.io/api => istio.io/api v0.0.0-20200911191701-0dc35ad5c478
	istio.io/client-go => istio.io/client-go v0.0.0-20200807182027-d287a5abb594
	k8s.io/api => k8s.io/api v0.19.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.0
	k8s.io/client-go => k8s.io/client-go v0.19.0
	k8s.io/code-generator => k8s.io/code-generator v0.19.0
	k8s.io/component-base => k8s.io/component-base v0.19.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.8.0
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.4.1
)
