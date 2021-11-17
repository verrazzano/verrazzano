// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
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
	err := adminSecretExists(c)
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
		return newAdminSecret(c, password)
	}

	log.Errorf("Rancher Post Install: Error checking availability of admin secret: %s", err)
	return err
}

// adminSecretExists creates the rancher admin secret if it does not exist
func adminSecretExists(c client.Client) error {
	if _, err := getAdminSecret(c); err != nil {
		return err
	}

	return nil
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

// patchAgents patches the cattle agents with the Rancher host and ip
func patchAgents(log *zap.SugaredLogger, c client.Client, host, ip string) error {
	hostAliases := []v1.HostAlias{
		{
			IP: ip,
			Hostnames: []string{
				host,
			},
		},
	}

	// Patch the agent Deployment if it exists
	deployment := &appsv1.Deployment{}
	deployNamespacedName := types.NamespacedName{Name: clusterAgentDeployName, Namespace: ComponentNamespace}
	if err := c.Get(context.TODO(), deployNamespacedName, deployment); err != nil {
		if apierrors.IsNotFound(err) {
			log.Infof("%v is not deployed, patching skipped", clusterAgentDeployName)
		} else {
			return err
		}
	} else {
		deployment.Spec.Template.Spec.HostAliases = hostAliases
		deploymentMerge := client.MergeFrom(deployment.DeepCopy())
		if err := c.Patch(context.TODO(), deployment, deploymentMerge); err != nil {
			return err
		}
	}

	// Patch the agent Daemonset if it exists
	daemonset := &appsv1.DaemonSet{}
	daemonsetNamespacedName := types.NamespacedName{Name: nodeAgentDaemonsetName, Namespace: ComponentNamespace}
	if err := c.Get(context.TODO(), daemonsetNamespacedName, daemonset); err != nil {
		if apierrors.IsNotFound(err) {
			log.Infof("%v is not deployed, patching skipped", nodeAgentDaemonsetName)
		} else {
			return err
		}
	} else {
		daemonset.Spec.Template.Spec.HostAliases = hostAliases
		daemonsetMerge := client.MergeFrom(daemonset.DeepCopy())
		if err := c.Patch(context.TODO(), daemonset, daemonsetMerge); err != nil {
			return err
		}
	}
	return nil
}

func getRancherIngressIP(log *zap.SugaredLogger, c client.Client) (string, error) {
	namespacedName := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      ComponentName,
	}
	log.Debugf("Getting %s ingress", ComponentName)
	ingress := &networking.Ingress{}
	if err := c.Get(context.TODO(), namespacedName, ingress); err != nil {
		return "", err
	}
	lbIngresses := ingress.Status.LoadBalancer.Ingress
	if len(lbIngresses) > 0 {
		return lbIngresses[0].IP, nil
	}
	return "", errors.New("load balancer IP not found")
}

func setServerURL(log *zap.SugaredLogger, password, hostname string) error {
	script := filepath.Join(config.GetInstallDir(), "put-rancher-server-url.sh")
	if _, stderr, err := bashFunc(script, password, hostname); err != nil {
		log.Errorf("Rancher post install: Failed to update Rancher server url: %s: %s", err, stderr)
		return err
	}
	return nil
}
