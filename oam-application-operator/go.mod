module github.com/verrazzano/verrazzano/oam-application-operator

go 1.13

require (
	github.com/Jeffail/gabs/v2 v2.2.0
	github.com/crossplane/crossplane-runtime v0.10.0
	github.com/crossplane/oam-kubernetes-runtime v0.3.2
	github.com/gertd/go-pluralize v0.1.7
	github.com/go-logr/logr v0.1.0
	github.com/golang/mock v1.4.4
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/stretchr/testify v1.5.1
	github.com/verrazzano/verrazzano-crd-generator v0.3.34
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	istio.io/api v0.0.0-20200629210345-933b83065c19
	istio.io/client-go v0.0.0-20200630182733-fd3f873f3f52
	k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/yaml v1.2.0
)

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8

replace k8s.io/client-go => k8s.io/client-go v0.18.6
