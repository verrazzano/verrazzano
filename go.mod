// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

module github.com/verrazzano/verrazzano

go 1.16

require (
	github.com/Jeffail/gabs/v2 v2.2.0
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/crossplane/crossplane-runtime v0.10.0
	github.com/crossplane/oam-kubernetes-runtime v0.3.2
	github.com/fatih/color v1.12.0 // indirect
	github.com/gertd/go-pluralize v0.1.7
	github.com/go-kit/kit v0.9.0
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0 // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.4.4
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/gordonklaus/ineffassign v0.0.0-20210104184537-8eed68eb605f
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v0.16.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.8
	github.com/jetstack/cert-manager v1.2.0
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/onsi/ginkgo/v2 v2.0.0-rc2
	github.com/onsi/gomega v1.17.0
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.30.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20210915214749-c084706c2272 // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	golang.org/x/net v0.0.0-20210917221730-978cfadd31cf // indirect
	golang.org/x/sys v0.0.0-20210917161153-d61c044b1678 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/tools v0.1.5
	google.golang.org/genproto v0.0.0-20210917145530-b395a37504d4 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
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
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.7.0
	go.uber.org/zap => go.uber.org/zap v1.16.0
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8
	k8s.io/api => k8s.io/api v0.19.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.2
	k8s.io/client-go => k8s.io/client-go v0.19.2
	k8s.io/code-generator => k8s.io/code-generator v0.19.2
)
