package helper_test_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHelperTest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HelperTest Suite")
}
