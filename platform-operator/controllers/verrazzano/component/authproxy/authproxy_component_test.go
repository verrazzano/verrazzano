package authproxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

func Test_authProxyComponent_IsReady(t *testing.T) {
	type fields struct {
		HelmComponent helm.HelmComponent
	}
	type args struct {
		ctx spi.ComponentContext
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := authProxyComponent{
				HelmComponent: tt.fields.HelmComponent,
			}
			assert.Equalf(t, tt.want, c.IsReady(tt.args.ctx), "IsReady(%v)", tt.args.ctx)
		})
	}
}
