package common

import (
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestManagedCluster(t *testing.T) {
	a := assert.New(t)
	cli := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects().Build()
	a.Nil(GetManagedClusterRegistrationSecret(cli))
}
