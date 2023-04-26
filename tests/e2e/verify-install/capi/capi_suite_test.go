package capi_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCapi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Capi Suite")
}
