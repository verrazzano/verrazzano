package apiconversion

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

const (
	ingressNGINXLabelValue = "ingress-nginx"
	ingressNGINXLabelKey   = "app.kubernetes.io/name"
)

type IngressNGINXReplicasModifier struct {
	replicas uint32
}

type IngressNGINXPodPerNodeAffinityModifier struct {
}

func (u IngressNGINXReplicasModifier) ModifyCR(cr *v1beta1.Verrazzano) {
	if cr.Spec.Components.IngressNGINX == nil {
		cr.Spec.Components.IngressNGINX = &v1beta1.IngressNginxComponent{}
	}
	cr.Spec.Components.IngressNGINX.ValueOverrides = fmt.Sprintf(`replicas: %v`, u.replicas)
}
