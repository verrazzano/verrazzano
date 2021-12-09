// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	resetPasswordCommand = "reset-password"
	ensureAdminCommand   = "ensure-default-admin"
)

func createAdminSecretIfNotExists(log *zap.SugaredLogger, c client.Client) error {
	_, err := common.GetAdminSecret(c)
	if err == nil {
		log.Debugf("Rancher Post Install: admin secret exists, skipping object creation")
		return nil
	}
	// if the admin secret doesn't exist, we need to create it
	if apierrors.IsNotFound(err) {
		password, resetPasswordErr := resetAdminPassword(c)
		if resetPasswordErr != nil {
			log.Errorf("Rancher Post Install: Failed to reset admin password: %s", resetPasswordErr)
			return resetPasswordErr
		}
		log.Debugf("Rancher Post Install: Creating new admin secret")
		return newAdminSecret(c, password)
	}

	log.Errorf("Rancher Post Install: Error checking admin secret availability: %s", err)
	return err
}

// retryResetPassword retries resetting the Rancher admin password using the Rancher shell
func resetAdminPassword(c client.Client) (string, error) {
	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return "", err
	}
	if err != nil {
		return "", err
	}
	podList := &v1.PodList{}
	labelMatcher := client.MatchingLabels{"app": common.RancherName}
	namespaceMatcher := client.InNamespace(common.CattleSystem)
	if err := c.List(context.TODO(), podList, namespaceMatcher, labelMatcher); err != nil {
		return "", err
	}
	if len(podList.Items) < 1 {
		return "", errors.New("no Rancher pods found")
	}
	pod := podList.Items[0]
	// Ensure the default rancer admin user is present
	_, stderr, err := k8sutil.ExecPod(cli, cfg, &pod, common.RancherName, []string{ensureAdminCommand})
	if err != nil {
		return "", fmt.Errorf("%s: %w", stderr, err)
	}
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, &pod, common.RancherName, []string{resetPasswordCommand})
	if err != nil {
		return "", err
	}
	// Shell output may have a trailing newline
	password := parsePasswordStdout(stdout)
	if password == "" {
		return "", fmt.Errorf("failed to reset Rancher admin password: %s", stderr)
	}
	return password, nil
}

// hack to parse the stdout of Rancher reset password
// we need to remove carriage returns and newlines from the stdout, since it is coming over from the pod's shell
// STDOUT is usually going to look something like this: "W1122 18:11:20.905585\nNew password for default admin user (user-p958n):\npassword\n"
func parsePasswordStdout(stdout string) string {
	partial := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	var password string
	switch len(partial) {
	case 3: // there may be three lines of stdout if a warning message is included
		password = partial[2]
	case 2: // usually there are two lines, the reset password message and the new password
		password = partial[1]
	default: // if there are not 2 or 3 lines, we cannot guess the output
		return ""
	}
	return strings.TrimSuffix(password, "\r")
}

// newAdminSecret generates the admin secret for Rancher
func newAdminSecret(c client.Client, password string) error {
	adminSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.CattleSystem,
			Name:      common.RancherAdminSecret,
		},
		Data: map[string][]byte{
			"password": []byte(password),
		},
	}
	return c.Create(context.TODO(), adminSecret)
}
