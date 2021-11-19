// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type AccessToken struct {
	Token string `json:"token"`
}

func createAdminSecretIfNotExists(log *zap.SugaredLogger, c client.Client) error {
	_, err := getAdminPassword(c)
	if err == nil {
		log.Infof("Rancher Post Install: admin secret exists, skipping object creation")
		return nil
	}
	// if the admin secret doesn't exist, we need to create it
	if apierrors.IsNotFound(err) {
		password, resetPasswordErr := resetAdminPassword(c)
		if resetPasswordErr != nil {
			log.Errorf("Rancher Post Install: Failed to reset admin password: %s", resetPasswordErr)
			return resetPasswordErr
		}
		log.Infof("Rancher Post Install: Creating new admin secret")
		return newAdminSecret(c, password)
	}

	log.Errorf("Rancher Post Install: Error checking admin secret availability: %s", err)
	return err
}

func getAdminPassword(c client.Client) (string, error) {
	adminSecret, err := getAdminSecret(c)
	if err != nil {
		return "", err
	}

	return string(adminSecret.Data["password"]), nil
}

func getAdminSecret(c client.Client) (*v1.Secret, error) {
	namespacedName := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      adminSecretName,
	}
	adminSecret := &v1.Secret{}
	err := c.Get(context.TODO(), namespacedName, adminSecret)
	return adminSecret, err
}

// retryResetPassword retries resetting the rancher admin password using the rancher shell
func resetAdminPassword(c client.Client) (string, error) {
	podList := &v1.PodList{}
	labelMatcher := client.MatchingLabels{"app": ComponentName}
	namespaceMatcher := client.InNamespace(ComponentNamespace)
	if err := c.List(context.TODO(), podList, namespaceMatcher, labelMatcher); err != nil {
		return "", err
	}
	if len(podList.Items) < 1 {
		return "", errors.New("no Rancher pods found")
	}
	podName := podList.Items[0].Name
	script := filepath.Join(config.GetInstallDir(), "reset-rancher-password.sh")
	stdout, stderr, err := bashFunc(script, podName, ComponentNamespace)
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, stderr)
	}
	// Shell output may have a trailing newline
	password := strings.TrimSuffix(stdout, "\n")
	if password == "" {
		return "", errors.New("failed to generate Rancher admin password, password is empty")
	}
	return password, nil
}

// newAdminSecret generates the admin secret for rancher
func newAdminSecret(c client.Client, password string) error {
	adminSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      adminSecretName,
		},
		Data: map[string][]byte{
			"password": []byte(password),
		},
	}
	return c.Create(context.TODO(), adminSecret)
}

func setServerURL(log *zap.SugaredLogger, password, hostname string) error {
	script := filepath.Join(config.GetInstallDir(), "put-rancher-server-url.sh")
	if _, stderr, err := bashFunc(script, password, hostname); err != nil {
		log.Errorf("Rancher post install: Failed to update Rancher server url: %s: %s", err, stderr)
		return err
	}
	return nil
}
