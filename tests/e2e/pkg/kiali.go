// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	networking "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	KialiName = "vmi-system-kiali"
)

func EventuallyGetKialiHost(cs *kubernetes.Clientset) string {
	var ingress *networking.Ingress
	gomega.Eventually(func() (*networking.Ingress, error) {
		var err error
		ingress, err = cs.NetworkingV1().Ingresses(kiali.ComponentNamespace).Get(context.TODO(), KialiName, v1.GetOptions{})
		return ingress, err
	}, waitTimeout, pollingInterval).ShouldNot(gomega.BeNil())
	rules := ingress.Spec.Rules
	gomega.Expect(len(rules)).To(gomega.Equal(1))
	gomega.Expect(rules[0].Host).To(gomega.ContainSubstring("kiali.vmi.system"))
	return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
}
