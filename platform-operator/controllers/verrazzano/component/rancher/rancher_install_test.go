package rancher

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

const (
	dnsSuffix = "DNS"
	name      = "NAME"
)

func TestAddAcmeIngressAnnotations(t *testing.T) {
	in := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}
	out := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/auth-realm":  fmt.Sprintf("%s auth", dnsSuffix),
				"external-dns.alpha.kubernetes.io/target": fmt.Sprintf("verrazzano-ingress.%s.%s", name, dnsSuffix),
				"cert-manager.io/issuer":                  "null",
				"cert-manager.io/issuer-kind":             "null",
				"external-dns.alpha.kubernetes.io/ttl":    "60",
			},
		},
	}

	addAcmeIngressAnnotations(name, dnsSuffix, &in)
	assert.Equal(t, out, in)
}

func TestAddCAIngressAnnotations(t *testing.T) {
	in := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}
	out := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/auth-realm": fmt.Sprintf("%s.%s auth", name, dnsSuffix),
				"cert-manager.io/cluster-issuer":         "verrazzano-cluster-issuer",
			},
		},
	}

	addCAIngressAnnotations(name, dnsSuffix, &in)
	assert.Equal(t, out, in)
}

func TestGetRancherContainer(t *testing.T) {
	var tests = []struct {
		in  []v1.Container
		out bool
	}{
		{[]v1.Container{{Name: "foo"}}, false},
		{[]v1.Container{{Name: "rancher"}}, true},
		{[]v1.Container{{Name: "baz"}, {Name: "rancher"}}, true},
		{[]v1.Container{{Name: "bar"}, {Name: "foo"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.in[0].Name, func(t *testing.T) {
			_, res := getRancherContainer(tt.in)
			assert.Equal(t, tt.out, res)
		})
	}
}
