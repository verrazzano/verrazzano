package rancher

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"testing"
)

func TestUseAdditionalCAs(t *testing.T) {
	var tests = []struct {
		in  vzapi.Acme
		out bool
	}{
		{vzapi.Acme{Environment: "dev"}, true},
		{vzapi.Acme{Environment: "production"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.in.Environment, func(t *testing.T) {
			assert.Equal(t, tt.out, useAdditionalCAs(tt.in))
		})
	}
}
