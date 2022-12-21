package security

import (
	"github.com/onsi/ginkgo/v2"
	"testing"
)

func TestSecurity(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Verrazzano Suite")
}
