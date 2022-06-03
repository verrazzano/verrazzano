package cmletsencrypt

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLetsEncryptCM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Let's Encrypt CM update Suite")
}
