// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/certificate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestWebhookInit(t *testing.T) {

	asserts := assert.New(t)
	logger := log.GetDebugEnabledLogger()
	caSecret := &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Immutable:  nil,
		Data:       nil,
		StringData: nil,
		Type:       "",
	}
	kubeClient := fake.NewSimpleClientset(caSecret)

	_, _, err := createExpectedCASecret(kubeClient)
	asserts.Nilf(err, "Unexpected error creating expected CA secret", err)

	wh, err := createExpectedValidatingWebhook(kubeClient, certificate.OperatorName)
	asserts.Nilf(err, "error should not be returned creating validation webhook configuration: %v", err)
	asserts.NotEmpty(wh)

	defer config.Set(config.OperatorConfig{VersionCheckEnabled: false})
	conf := config.Get()

	err = WebhookInit(conf, logger)
	asserts.NoError(err)

}
