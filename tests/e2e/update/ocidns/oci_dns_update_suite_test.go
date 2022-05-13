package ocidns

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func OCIDNS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Update OCI DNS Suite")
}
