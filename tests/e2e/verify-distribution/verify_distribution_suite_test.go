package verify_distribution_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVerifyDistribution(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VerifyDistribution Suite")
}
