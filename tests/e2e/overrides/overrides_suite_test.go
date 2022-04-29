package overrides

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOverrides(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Overrides Suite")
}
