// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"errors"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	os2 "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// CRI-O does not deliver MKNOD by default, until https://github.com/rancher/rancher/pull/27582 is merged we must add the capability
func patchRancherDeployment(c client.Client) error {
	deployment := appsv1.Deployment{}
	namespacedName := types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}
	if err := c.Get(context.TODO(), namespacedName, &deployment); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	deploymentMerge := client.MergeFrom(deployment.DeepCopy())
	containers := deployment.Spec.Template.Spec.Containers
	container, ok := getRancherContainer(containers)
	if !ok {
		return errors.New("rancher container was not found")
	}
	container.SecurityContext = &v1.SecurityContext{
		Capabilities: &v1.Capabilities{
			Add: []v1.Capability{"MKNOD"},
		},
	}

	if err := c.Patch(context.TODO(), &deployment, deploymentMerge); err != nil {
		return err
	}

	return nil
}

func getRancherContainer(containers []v1.Container) (v1.Container, bool) {
	for _, container := range containers {
		if container.Name == ComponentName {
			return container, true
		}
	}

	return v1.Container{}, false
}

//patchRancherIngress patches the rancher ingress with certificate annotations to support TLS
func patchRancherIngress(c client.Client, vz *vzapi.Verrazzano) error {
	cm := vz.Spec.Components.CertManager
	if cm == nil {
		return errors.New("CertificateManager was not found in the effective CR")
	}
	dnsSuffix, err := nginx.GetDNSSuffix(c, vz)
	if err != nil {
		return err
	}
	namespacedName := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      ComponentName,
	}
	ingress := &networking.Ingress{}
	if err := c.Get(context.TODO(), namespacedName, ingress); err != nil {
		return err
	}
	ingressMerge := client.MergeFrom(ingress.DeepCopy())
	ingress.Annotations["kubernetes.io/tls-acme"] = "true"
	if (cm.Certificate.Acme != vzapi.Acme{}) {
		addAcmeIngressAnnotations(vz.Spec.EnvironmentName, dnsSuffix, ingress)
	} else {
		addCAIngressAnnotations(vz.Spec.EnvironmentName, dnsSuffix, ingress)
	}
	return c.Patch(context.TODO(), ingress, ingressMerge)
}

//addAcmeIngressAnnotations annotate ingress with ACME specific values
func addAcmeIngressAnnotations(name, dnsSuffix string, ingress *networking.Ingress) {
	ingress.Annotations["nginx.ingress.kubernetes.io/auth-realm"] = fmt.Sprintf("%s auth", dnsSuffix)
	ingress.Annotations["external-dns.alpha.kubernetes.io/target"] = fmt.Sprintf("verrazzano-ingress.%s.%s", name, dnsSuffix)
	ingress.Annotations["cert-manager.io/issuer"] = "null"
	ingress.Annotations["cert-manager.io/issuer-kind"] = "null"
	ingress.Annotations["external-dns.alpha.kubernetes.io/ttl"] = "60"
}

//addCAIngressAnnotations annotate ingress with custom CA specific values
func addCAIngressAnnotations(name, dnsSuffix string, ingress *networking.Ingress) {
	ingress.Annotations["nginx.ingress.kubernetes.io/auth-realm"] = fmt.Sprintf("%s.%s auth", name, dnsSuffix)
	ingress.Annotations["cert-manager.io/cluster-issuer"] = "verrazzano-cluster-issuer"
}

func waitForRancherReady(log *zap.SugaredLogger) error {
	if _, stderr, err := run(log, "kubectl", "-n cattle-system rollout status -w deploy/rancher"); err != nil {
		return errors.New(stderr)
	}
	if _, stderr, err := run(log, "kubectl", "wait --for=condition=ready cert tls-rancher-ingress -n cattle-system"); err != nil {
		return errors.New(stderr)
	}
	return nil
}

func createAdminSecretIfNotExists(log *zap.SugaredLogger, c client.Client) error {
	err, ok := adminSecretExists(c)
	if ok {
		log.Infof("Rancher admin secret exists, skipping admin secret creation")
		return nil
	}
	// if the admin secret doesn't exist, we need to create it
	if apierrors.IsNotFound(err) {
		password, err := resetAdminPassword(log, c)
		if err != nil {
			return err
		}
		return newAdminSecret(c, password)
	}

	return err
}

// adminSecretExists creates the rancher admin secret if it does not exist
func adminSecretExists(c client.Client) (error, bool) {
	if _, err := getAdminSecret(c); err != nil {
		return err, false
	} else {
		return nil, true
	}
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
		Name:      AdminSecret,
	}
	adminSecret := &v1.Secret{}
	err := c.Get(context.TODO(), namespacedName, adminSecret)
	return adminSecret, err
}

// retryResetPassword retries resetting the rancher admin password using the rancher shell
func resetAdminPassword(log *zap.SugaredLogger, c client.Client) (string, error) {
	podMeta := &metav1.PartialObjectMetadataList{}
	labelMatcher := client.MatchingLabels{"app": "rancher"}
	namespaceMatcher := client.InNamespace(ComponentNamespace)
	if err := c.List(context.TODO(), podMeta, namespaceMatcher, labelMatcher); err != nil {
		return "", err
	}
	if len(podMeta.Items) < 1 {
		log.Errorf("Rancher post install: Failed to reset admin password, no pods found")
		return "", errors.New("failed to reset admin password")
	}
	podName := podMeta.Items[0].Name
	stdout, stderr, err := os2.RunBash("kubectl", "exec", podName, "-n", ComponentNamespace, "--", "reset-password", "|", "tail", "-1")
	if err != nil {
		log.Errorf("Rancher post install: Failed to reset admin password, %s: %s", err, stderr)
	}
	return stdout, nil
}

// newAdminSecret generates the admin secret for rancher
func newAdminSecret(c client.Client, password string) error {
	adminSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      AdminSecret,
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
		v1.HostAlias{
			IP: ip,
			Hostnames: []string{
				host,
			},
		},
	}

	// Patch the agent Deployment if it exists
	deployment := &appsv1.Deployment{}
	deployNamespacedName := types.NamespacedName{Name: clusterAgentDeploy, Namespace: ComponentNamespace}
	if err := c.Get(context.TODO(), deployNamespacedName, deployment); err != nil {
		if apierrors.IsNotFound(err) {
			log.Infof("%v is not deployed, patching skipped", clusterAgentDeploy)
		} else {
			return err
		}
	}
	deployment.Spec.Template.Spec.HostAliases = hostAliases
	deploymentMerge := client.MergeFrom(deployment.DeepCopy())
	if err := c.Patch(context.TODO(), deployment, deploymentMerge); err != nil {
		return err
	}

	// Patch the agent Daemonset if it exists
	daemonset := &appsv1.DaemonSet{}
	daemonsetNamespacedName := types.NamespacedName{Name: nodeAgentDaemonset, Namespace: ComponentNamespace}
	if err := c.Get(context.TODO(), daemonsetNamespacedName, daemonset); err != nil {
		if apierrors.IsNotFound(err) {
			log.Infof("%v is not deployed, patching skipped", nodeAgentDaemonset)
		} else {
			return err
		}
	}
	daemonset.Spec.Template.Spec.HostAliases = hostAliases
	daemonsetMerge := client.MergeFrom(daemonset.DeepCopy())
	if err := c.Patch(context.TODO(), daemonset, daemonsetMerge); err != nil {
		return err
	}
	return nil
}

func getRancherIngressIp(log *zap.SugaredLogger, c client.Client) (string, error) {
	attempts := 10
	wait := 5 * time.Second
	namespacedName := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      ComponentName,
	}
	log.Debugf("Getting %s ingress", ComponentName)
	ingress := &networking.Ingress{}
	for attempt := 0; attempt < attempts; attempt++ {
		if err := c.Get(context.TODO(), namespacedName, ingress); err != nil {
			return "", err
		}
		lbIngresses := ingress.Status.LoadBalancer.Ingress
		if len(lbIngresses) > 0 {
			return lbIngresses[0].IP, nil
		}
		time.Sleep(wait)
	}

	return "", errors.New("timeout waiting for Rancher loadbalancer IP")
}

func setServerUrl(log *zap.SugaredLogger, password, hostname string) error {
	script := filepath.Join(config.GetInstallDir(), "put-rancher-server-url.sh")
	if _, stderr, err := os2.RunBash(script, password, hostname); err != nil {
		log.Errorf("Rancher post install: Failed to update Rancher server url: %s: %s", err, stderr)
		return err
	}
	return nil
}
