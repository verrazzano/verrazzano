// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

module github.com/verrazzano/verrazzano

go 1.20

require (
	github.com/Jeffail/gabs/v2 v2.6.1
	github.com/cert-manager/cert-manager v1.9.1
	github.com/crossplane/crossplane-runtime v0.17.0
	github.com/crossplane/oam-kubernetes-runtime v0.3.3
	github.com/fatih/camelcase v1.0.0
	github.com/fvbommel/sortorder v1.0.1
	github.com/gertd/go-pluralize v0.2.0
	github.com/go-logr/logr v1.2.3
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.5.9
	github.com/google/uuid v1.3.0
	github.com/gordonklaus/ineffassign v0.0.0-20210104184537-8eed68eb605f
	github.com/hashicorp/go-retryablehttp v0.6.8
	github.com/mattn/go-isatty v0.0.16
	github.com/mitchellh/go-testing-interface v1.0.0
	github.com/onsi/ginkgo/v2 v2.1.6
	github.com/onsi/gomega v1.20.1
	github.com/oracle/oci-go-sdk/v53 v53.1.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.59.1
	github.com/prometheus/client_golang v1.12.1
	github.com/spf13/cobra v1.6.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.3
	github.com/verrazzano/verrazzano-monitoring-operator v0.0.31-0.20230201202534-ac5ebe880e95
	go.uber.org/zap v1.21.0
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	golang.org/x/text v0.13.0
	golang.org/x/tools v0.7.0
	google.golang.org/protobuf v1.30.0
	gopkg.in/yaml.v3 v3.0.1
	helm.sh/helm/v3 v3.10.3
	istio.io/api v0.0.0-20221208152505-d807bc07da6a
	istio.io/client-go v1.15.4
	k8s.io/api v0.25.2
	k8s.io/apiextensions-apiserver v0.25.2
	k8s.io/apimachinery v0.25.2
	k8s.io/cli-runtime v0.25.2
	k8s.io/client-go v0.25.2
	k8s.io/code-generator v0.25.2
	k8s.io/kube-openapi v0.0.0-20220803164354-a70c9af30aea
	k8s.io/kubectl v0.25.2
	sigs.k8s.io/cluster-api v1.2.0
	sigs.k8s.io/controller-runtime v0.12.3
	sigs.k8s.io/controller-tools v0.9.2
	sigs.k8s.io/kind v0.14.0
	sigs.k8s.io/yaml v1.3.0
)

require (
	github.com/BurntSushi/toml v1.1.0 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.0 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v1.4.10 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/drone/envsubst/v2 v2.0.0-20210730161058-179042472c46 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-logr/zapr v1.2.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.1 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/gobuffalo/flect v0.2.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/cel-go v0.12.6 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-github/v45 v45.2.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.3.3 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sony/gobreaker v0.4.2-0.20210216022020-dd874f9dd33b // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.12.0 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/subosito/gotenv v1.3.0 // indirect
	github.com/valyala/fastjson v1.6.3 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	go.starlark.net v0.0.0-20200306205701-8dd3e2ee1dd5 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/oauth2 v0.7.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/term v0.13.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.66.4 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apiserver v0.25.2 // indirect
	k8s.io/cluster-bootstrap v0.24.0 // indirect
	k8s.io/component-base v0.25.2 // indirect
	k8s.io/gengo v0.0.0-20220902162205-c0856e24416d // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-aggregator v0.24.2 // indirect
	k8s.io/utils v0.0.0-20230209194617-a36077c30491 // indirect
	sigs.k8s.io/gateway-api v0.4.3 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kustomize/api v0.12.1 // indirect
	sigs.k8s.io/kustomize/kyaml v0.13.9 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)

require (
	cloud.google.com/go/compute v1.19.1 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	github.com/alessio/shellescape v1.4.1 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	golang.org/x/mod v0.9.0 // indirect
)

replace (
	github.com/ajeddeloh/go-json v0.0.0-20200220154158-5ae607161559 => github.com/coreos/go-json v0.0.0-20220325222439-31b2177291ae
	github.com/crossplane/crossplane-runtime => github.com/verrazzano/crossplane-runtime v0.17.0-1
	github.com/crossplane/oam-kubernetes-runtime => github.com/verrazzano/oam-kubernetes-runtime v0.3.3-5
	github.com/docker/docker => github.com/docker/docker v20.10.24+incompatible
	github.com/emicklei/go-restful => github.com/emicklei/go-restful v2.16.0+incompatible
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	github.com/onsi/ginkgo/v2 => github.com/onsi/ginkgo/v2 v2.0.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.17.0
	github.com/spf13/cobra => github.com/spf13/cobra v1.6.1
	github.com/stretchr/testify => github.com/stretchr/testify v1.7.1
	golang.org/x/lint => golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5
	golang.org/x/net => golang.org/x/net v0.17.0
	google.golang.org/grpc => google.golang.org/grpc v1.56.3
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.1
	helm.sh/helm/v3 => helm.sh/helm/v3 v3.10.3
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.11.2
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.8.0
	sigs.k8s.io/kind => github.com/verrazzano/kind v0.0.0-20221129215948-885481909133
)

require (
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d
	github.com/mitchellh/go-wordwrap v1.0.0
	github.com/russross/blackfriday v1.5.2
)
