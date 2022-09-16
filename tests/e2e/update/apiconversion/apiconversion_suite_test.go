package apiconversion_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestApiconversion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Apiconversion Suite")
}
