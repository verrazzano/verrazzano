package none_profile

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestNoneProfile(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "None Profile Suite")
}
