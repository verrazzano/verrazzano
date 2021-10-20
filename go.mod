// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

module github.com/verrazzano/verrazzano

go 1.16

require (
	github.com/Jeffail/gabs/v2 v2.6.1
	github.com/crossplane/crossplane-runtime v0.15.1
	github.com/gertd/go-pluralize v0.1.7
	github.com/go-logr/logr v0.4.0
	github.com/golang/mock v1.5.0
	github.com/gordonklaus/ineffassign v0.0.0-20210914165742-4cc7213b9bc8
	github.com/hashicorp/go-retryablehttp v0.7.0
	github.com/jetstack/cert-manager v1.5.4
	github.com/joshdk/go-junit v0.0.0-20210226021600-6145f504ca0d
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.14.0
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/verrazzano/oam-kubernetes-runtime v0.3.3-0.20211020210605-352fb58ac665
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
