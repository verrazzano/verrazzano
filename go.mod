// Copyright (c) 2020, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

module github.com/verrazzano/verrazzano

go 1.21

require (
	github.com/Jeffail/gabs/v2 v2.7.0
	github.com/cert-manager/cert-manager v1.9.1
	github.com/coreos/go-oidc/v3 v3.6.0
	github.com/crossplane/crossplane-runtime v0.17.0
	github.com/crossplane/oam-kubernetes-runtime v0.3.3
	github.com/fatih/camelcase v1.0.0
	github.com/fluent/fluent-operator/v2 v2.3.0
	github.com/fvbommel/sortorder v1.0.1
	github.com/gertd/go-pluralize v0.2.0
	github.com/go-logr/logr v1.2.4
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.5.9
	github.com/google/uuid v1.3.0
	github.com/gordonklaus/ineffassign v0.0.0-20210104184537-8eed68eb605f
	github.com/hashicorp/go-retryablehttp v0.7.4
	github.com/mattn/go-isatty v0.0.16
	github.com/mitchellh/go-testing-interface v1.0.0
	github.com/onsi/ginkgo/v2 v2.9.1
	github.com/onsi/gomega v1.27.7
	github.com/oracle/oci-go-sdk/v53 v53.1.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.64.1
	github.com/prometheus/client_golang v1.14.0
	github.com/spf13/cobra v1.6.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.3
	github.com/verrazzano/pkg v0.0.2
	github.com/verrazzano/verrazzano-modules v0.0.0-20230915154150-1fe7062cbccd
	github.com/verrazzano/verrazzano-monitoring-operator v0.0.31-0.20230425042339-1243c1ab0595
	go.uber.org/zap v1.24.0
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	golang.org/x/text v0.13.0
	golang.org/x/tools v0.7.0
	google.golang.org/protobuf v1.30.0
	gopkg.in/yaml.v3 v3.0.1
	helm.sh/helm/v3 v3.10.3
	istio.io/api v0.0.0-20230712174848-a2b2de508c88
	istio.io/client-go v1.17.4
	k8s.io/api v0.26.3
	k8s.io/apiextensions-apiserver v0.26.1
	k8s.io/apimachinery v0.27.2
	k8s.io/cli-runtime v0.25.2
	k8s.io/client-go v0.26.3
	k8s.io/code-generator v0.25.4
	k8s.io/kube-openapi v0.0.0-20230501164219-8b0f38b5fd1f
	k8s.io/kubectl v0.25.2
	sigs.k8s.io/cluster-api v1.3.3
	sigs.k8s.io/controller-runtime v0.14.6
	sigs.k8s.io/controller-tools v0.9.2
	sigs.k8s.io/yaml v1.3.0
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/BurntSushi/toml v1.1.0 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.1
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/Masterminds/squirrel v1.5.3 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v1.4.10 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/containerd/containerd v1.6.6 // indirect
	github.com/cyphar/filepath-securejoin v0.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/cli v20.10.17+incompatible // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/docker/docker v20.10.17+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.6.4 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/drone/envsubst/v2 v2.0.0-20210730161058-179042472c46 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-gorp/gorp/v3 v3.0.2 // indirect
	github.com/go-logr/zapr v1.2.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.1 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/gobuffalo/flect v0.3.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
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
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/gosuri/uitable v0.0.4 // indirect
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.3.3 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/jmoiron/sqlx v1.3.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/lib/pq v1.10.6 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/term v0.0.0-20220808134915-39b0c02b01ae // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.3-0.20211202183452-c5a74bcca799 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.5 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/rivo/uniseg v0.4.2 // indirect
	github.com/rubenv/sql-migrate v1.1.2 // indirect
	github.com/russross/blackfriday v1.5.2 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/sony/gobreaker v0.4.2-0.20210216022020-dd874f9dd33b // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.13.0 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/subosito/gotenv v1.4.1 // indirect
	github.com/valyala/fastjson v1.6.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	go.etcd.io/etcd/api/v3 v3.5.5 // indirect
	go.starlark.net v0.0.0-20200306205701-8dd3e2ee1dd5 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/crypto v0.14.0
	golang.org/x/net v0.10.0
	golang.org/x/oauth2 v0.7.0
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/term v0.13.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.54.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apiserver v0.26.1 // indirect
	k8s.io/cluster-bootstrap v0.25.0 // indirect
	k8s.io/component-base v0.26.1 // indirect
	k8s.io/gengo v0.0.0-20220902162205-c0856e24416d // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-aggregator v0.24.2 // indirect
	k8s.io/utils v0.0.0-20230209194617-a36077c30491
	oras.land/oras-go v1.2.0 // indirect
	sigs.k8s.io/gateway-api v0.4.3 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kustomize/api v0.12.1 // indirect
	sigs.k8s.io/kustomize/kyaml v0.13.9 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)

require (
	cloud.google.com/go/compute v1.19.1 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	github.com/go-jose/go-jose/v3 v3.0.0 // indirect
	github.com/go-openapi/errors v0.20.3 // indirect
	golang.org/x/mod v0.9.0 // indirect
)

replace (
	cloud.google.com/go/compute => cloud.google.com/go/compute v1.19.1
	github.com/ajeddeloh/go-json v0.0.0-20200220154158-5ae607161559 => github.com/coreos/go-json v0.0.0-20220325222439-31b2177291ae
	github.com/cespare/xxhash/v2 => github.com/cespare/xxhash/v2 v2.2.0
	github.com/containerd/containerd => github.com/containerd/containerd v1.6.18
	github.com/crossplane/crossplane-runtime => github.com/verrazzano/crossplane-runtime v0.17.0-1
	github.com/crossplane/oam-kubernetes-runtime => github.com/verrazzano/oam-kubernetes-runtime v0.3.3-5
	github.com/cyphar/filepath-securejoin => github.com/cyphar/filepath-securejoin v0.2.4
	github.com/docker/distribution => github.com/docker/distribution v2.8.2+incompatible
	github.com/docker/docker => github.com/docker/docker v20.10.24+incompatible
	github.com/emicklei/go-restful => github.com/emicklei/go-restful v2.16.0+incompatible
	github.com/fluent/fluent-operator/v2 => github.com/verrazzano/fluent-operator/v2 v2.2.0
	github.com/go-logr/logr => github.com/go-logr/logr v1.2.3
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	github.com/matttproud/golang_protobuf_extensions => github.com/matttproud/golang_protobuf_extensions v1.0.4
	github.com/onsi/ginkgo/v2 => github.com/onsi/ginkgo/v2 v2.0.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.17.0
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.13.0
	github.com/spf13/cobra => github.com/spf13/cobra v1.6.1
	github.com/stretchr/testify => github.com/stretchr/testify v1.7.1
	go.uber.org/zap => go.uber.org/zap v1.21.0
	golang.org/x/crypto => golang.org/x/crypto v0.14.0
	golang.org/x/lint => golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5
	golang.org/x/net => golang.org/x/net v0.17.0
	golang.org/x/oauth2 => golang.org/x/oauth2 v0.7.0
	golang.org/x/sys => golang.org/x/sys v0.13.0
	golang.org/x/term => golang.org/x/term v0.13.0
	golang.org/x/text => golang.org/x/text v0.13.0
	golang.org/x/tools => golang.org/x/tools v0.1.12
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1
	google.golang.org/grpc => google.golang.org/grpc v1.56.3
	google.golang.org/protobuf => google.golang.org/protobuf v1.30.0
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.1
	helm.sh/helm/v3 => helm.sh/helm/v3 v3.10.3
	k8s.io/api => k8s.io/api v0.25.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.25.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.25.4
	k8s.io/client-go => k8s.io/client-go v0.25.4
	k8s.io/code-generator => k8s.io/code-generator v0.25.4
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20220803164354-a70c9af30aea
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.13.1
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.8.0
	sigs.k8s.io/kind => github.com/verrazzano/kind v0.0.0-20221129215948-885481909133
)
