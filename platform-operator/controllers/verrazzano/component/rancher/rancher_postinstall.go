// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	resetPasswordCommand = "reset-password"
	ensureAdminCommand   = "ensure-default-admin"
	BootstrapSecret      = "bootstrap-secret"
)

func createAdminSecretIfNotExists(log vzlog.VerrazzanoLogger, c client.Client) error {
	_, err := common.GetAdminSecret(c)
	if err == nil {
		log.Debugf("Rancher Post Install: admin secret exists, skipping object creation")
		return nil
	}
	// if the admin secret doesn't exist, we need to create it
	if apierrors.IsNotFound(err) {
		password, resetPasswordErr := resetAdminPassword(c)
		if resetPasswordErr != nil {
			return log.ErrorfNewErr("Failed to reset Rancher admin password: %v", resetPasswordErr)
		}
		log.Debugf("Rancher Post Install: Creating new admin secret")
		return newAdminSecret(c, password)
	}

	return log.ErrorfNewErr("Failed checking Rancher admin secret availability: %v", err)
}

func removeBootstrapSecretIfExists(log vzlog.VerrazzanoLogger, c client.Client) error {
	secret := &v1.Secret{}
	nsName := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      BootstrapSecret}

	// check if the secret exists
	if err := c.Get(context.TODO(), nsName, secret); err != nil {
		// if it does not, there is nothing to do and no error, so just return
		return nil
	}
	if err := c.Delete(context.TODO(), secret); err != nil {
		return err
	}
	log.Debugf("Deleted Rancher bootstrap secret")
	// worked fine, return nil
	return nil
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
	pod := selectFirstReadyPod(podList.Items)
	if pod == nil {
		return "", fmt.Errorf("Failed to reset Rancher admin secret, no running Rancher pods found")
	}
	// Ensure the default Rancher admin user is present
	_, stderr, err := k8sutil.ExecPod(cli, cfg, pod, common.RancherName, []string{ensureAdminCommand})
	if err != nil {
		return "", fmt.Errorf("Failed execing into Rancher pod %s: %v", stderr, err)
	}
	stdout, stderr, err := k8sutil.ExecPod(cli, cfg, pod, common.RancherName, []string{resetPasswordCommand})
	if err != nil {
		return "", err
	}
	// Shell output may have a trailing newline
	password := parsePasswordStdout(stdout)
	if password == "" {
		return "", fmt.Errorf("Failed to reset Rancher admin password: %s", stderr)
	}
	return password, nil
}

// selectFirstReadyPod selects the first running pod from the slice and return a pointer, nil if none found
func selectFirstReadyPod(pods []v1.Pod) *v1.Pod {
	for _, pod := range pods {
		if isPodReady(pod) {
			return &pod
		}
	}
	return nil
}

// isPodReady determines if the pod is running by checking for a Ready condition with Status equal True
func isPodReady(pod v1.Pod) bool {
	conditions := pod.Status.Conditions
	for j := range conditions {
		if conditions[j].Type == "Ready" && conditions[j].Status == "True" {
			return true
		}
	}
	return false
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

// addNameSpaceLabels labels the namespace created by rancher component
func labelNamespace(c client.Client) error {
	nsList := &v1.NamespaceList{}
	err := c.List(context.TODO(), nsList, &client.ListOptions{})
	if err != nil {
		return err
	}
	for i := range nsList.Items {
		if _, found := nsList.Items[i].Labels[namespaceLabelKey]; isRancherNamespace(&(nsList.Items[i])) && !found {
			nsList.Items[i].Labels[namespaceLabelKey] = nsList.Items[i].Name
			c.Update(context.TODO(), &(nsList.Items[i]), &client.UpdateOptions{})
		}
	}
	return nil
}
