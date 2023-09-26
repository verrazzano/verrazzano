// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhook

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/k8s/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	adminv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlcli "sigs.k8s.io/controller-runtime/pkg/client"
)

func DeleteMutatingWebhookConfiguration(log vzlog.VerrazzanoLogger, client ctrlcli.Client, names []string) error {
	for _, name := range names {
		wh := adminv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}
		log.Progressf("Deleting MutatingWebhookConfiguration %s", name)
		if err := client.Delete(context.TODO(), &wh); err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return log.ErrorfThrottledNewErr("Failed to delete MutatingWebhookConfiguration %:, %v", name, err)
		}
	}
	return nil
}

func DeleteValidatingWebhookConfiguration(log vzlog.VerrazzanoLogger, client ctrlcli.Client, name string) error {
	wh := adminv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	log.Progressf("Deleting ValidatingWebhookConfiguration %s", name)
	if err := client.Delete(context.TODO(), &wh); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return log.ErrorfThrottledNewErr("Failed to delete ValidatingWebhookConfiguration %s: %v", name, err)
	}
	return nil
}
