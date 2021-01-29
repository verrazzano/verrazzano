// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

module github.com/verrazzano/verrazzano/tests

go 1.13

require (
	github.com/golang/mock v1.4.4 // indirect
	github.com/gordonklaus/ineffassign v0.0.0-20210104184537-8eed68eb605f // indirect
	github.com/hashicorp/go-retryablehttp v0.6.8
	github.com/jetstack/cert-manager v1.1.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/verrazzano/verrazzano-monitoring-operator v0.0.25
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	golang.org/x/mod v0.4.1 // indirect
	golang.org/x/tools v0.1.0 // indirect
	istio.io/api v0.0.0-20200629210345-933b83065c19
	istio.io/client-go v0.0.0-20200630182733-fd3f873f3f52
	k8s.io/api v0.19.0
	k8s.io/apiextensions-apiserver v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.3.1
	k8s.io/api => k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.6
	k8s.io/client-go => k8s.io/client-go v0.18.6
)
