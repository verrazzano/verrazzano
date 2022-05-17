// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Constants for Kubernetes resource names
const (
	// note: VZ-5241 In Rancher 2.6.3 the agent was moved from cattle-fleet-system ns
	// to a new cattle-fleet-local-system ns, the rancher-operator-system ns was
	// removed, and the rancher-operator is no longer deployed
	FleetSystemNamespace      = "cattle-fleet-system"
	FleetLocalSystemNamespace = "cattle-fleet-local-system"
	defaultSecretNamespace    = "cert-manager"
	namespaceLabelKey         = "verrazzano.io/namespace"
	rancherTLSSecretName      = "tls-ca"
	defaultVerrazzanoName     = "verrazzano-ca-certificate-secret"
	fleetAgentDeployment      = "fleet-agent"
	fleetControllerDeployment = "fleet-controller"
	gitjobDeployment          = "gitjob"
	rancherWebhookDeployment  = "rancher-webhook"
)

// Helm Chart setter keys
const (
	ingressTLSSourceKey     = "ingress.tls.source"
	additionalTrustedCAsKey = "additionalTrustedCAs"
	privateCAKey            = "privateCA"

	// Rancher registry Keys
	useBundledSystemChartKey = "useBundledSystemChart"
	systemDefaultRegistryKey = "systemDefaultRegistry"

	// LE Keys
	letsEncryptIngressClassKey = "letsEncrypt.ingress.class"
	letsEncryptEmailKey        = "letsEncrypt.email"
	letsEncryptEnvironmentKey  = "letsEncrypt.environment"
)

const (
	letsEncryptTLSSource       = "letsEncrypt"
	caTLSSource                = "secret"
	caCertsPem                 = "cacerts.pem"
	caCert                     = "ca.crt"
	privateCAValue             = "true"
	useBundledSystemChartValue = "true"
)

func useAdditionalCAs(acme vzapi.Acme) bool {
	return acme.Environment != "production"
}

func getRancherHostname(c client.Client, vz *vzapi.Verrazzano) (string, error) {
	dnsSuffix, err := vzconfig.GetDNSSuffix(c, vz)
	if err != nil {
		return "", err
	}
	rancherHostname := fmt.Sprintf("%s.%s.%s", common.RancherName, vz.Spec.EnvironmentName, dnsSuffix)
	return rancherHostname, nil
}

// isRancherReady checks that the Rancher component is in a 'Ready' state, as defined
// in the body of this function
func isRancherReady(ctx spi.ComponentContext) bool {
	log := ctx.Log()
	c := ctx.Client()

	// Temporary work around for Rancher issue 36914
	err := checkRancherUpgradeFailure(c, log)
	if err != nil {
		log.ErrorfThrottled("Error checking Rancher pod logs: %s", err.Error())
		return false
	}

	deployments := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
		{
			Name:      rancherWebhookDeployment,
			Namespace: ComponentNamespace,
		},
		{
			Name:      fleetAgentDeployment,
			Namespace: FleetLocalSystemNamespace,
		},
		{
			Name:      fleetControllerDeployment,
			Namespace: FleetSystemNamespace,
		},
		{
			Name:      gitjobDeployment,
			Namespace: FleetSystemNamespace,
		},
	}

	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(log, c, deployments, 1, prefix)
}

// checkRancherUpgradeFailure - temporary work around for Rancher issue 36914. During an upgrade, the Rancher pods
// are recycled.  When the leader pod is restarted, it is possible that a Rancher 2.5.9 pod could
// acquire leader and recreate the downloaded the helm charts it's requires.
//
// If one of the Rancher pods is failing to find the rancher-webhook, recycle that pod.
func checkRancherUpgradeFailure(c client.Client, log vzlog.VerrazzanoLogger) error {
	ctx := context.TODO()

	// Get the Rancher pods
	podList := &corev1.PodList{}
	err := c.List(ctx, podList, client.InNamespace(ComponentNamespace), client.MatchingLabels{"app": "rancher"})
	if err != nil {
		return err
	}
	if len(podList.Items) == 0 {
		return nil
	}

	config, err := ctrl.GetConfig()
	if err != nil {
		return err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// Check the logs of each pod
	for i, pod := range podList.Items {
		// Skip pods that are not ready, they will get checked again in another call to isReady.
		if !isPodReady(pod) {
			break
		}

		// Get the pod log stream
		logStream, err := clientSet.CoreV1().Pods(ComponentNamespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: "rancher"}).Stream(ctx)
		if err != nil {
			return err
		}
		defer logStream.Close()

		// Search the stream for the expected text
		restartPod := false
		scanner := bufio.NewScanner(logStream)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "Failed to find system chart rancher-webhook will try again in 5 seconds") {
				restartPod = true
				break
			}
		}

		// If the pod is failing to find the system chart for rancher-webhook, the wrong helm charts are
		// being used by the Rancher pod. Restart the pod. This will cause another Rancher pod to become the leader,
		// and, if needed, will recreate the custom resources related to the helm charts.
		if restartPod {
			// Delete custom resources containing helm charts to use
			err := deleteClusterRepos(log)
			if err != nil {
				return err
			}
			log.Infof("Rancher IsReady: Restarting pod %s", pod.Name)
			err = c.Delete(ctx, &podList.Items[i])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// deleteClusterRepos - temporary work around for Rancher issue 36914. On upgrade of Rancher
// the setting of useBundledSystemChart does not appear to be honored, and the downloaded
// helm charts for the previous release of Rancher are used (instead of the charts on the Rancher
// container image).
func deleteClusterRepos(log vzlog.VerrazzanoLogger) error {

	config, err := ctrl.GetConfig()
	if err != nil {
		log.Debugf("Rancher Pre-Upgrade: Failed getting config: %v", err)
		return err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Debugf("Rancher Pre-Upgrade: Failed creating dynamic client: %v", err)
		return err
	}

	// Configure the GVR
	gvr := schema.GroupVersionResource{
		Group:    "management.cattle.io",
		Version:  "v3",
		Resource: "settings",
	}

	// Get the name of the default branch for the helm charts
	name := "chart-default-branch"
	chartDefaultBranch, err := dynamicClient.Resource(gvr).Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		log.Debugf("Rancher IsReady: Failed getting settings.management.cattle.io %s: %v", name, err)
		return err
	}

	// Obtain the name of the default branch from the custom resource
	defaultBranch, _, err := unstructured.NestedString(chartDefaultBranch.Object, "default")
	if err != nil {
		log.Debugf("Rancher IsReady: Failed to find default branch value in settings.management.cattle.io %s: %v", name, err)
		return err
	}

	// Only need to delete the custom resources if the default branch is still release-2.5
	log.Infof("Rancher Pre-Upgrade: The default release branch is currently set to %s", defaultBranch)
	if defaultBranch != "release-v2.5" {
		return nil
	}

	// Delete settings.management.cattle.io chart-default-branch
	err = dynamicClient.Resource(gvr).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Debugf("Rancher Pre-Upgrade: Failed deleting settings.management.cattle.io %s: %v", name, err)
		return err
	}
	log.Infof("Rancher Pre-Upgrade: Deleted settings.management.cattle.io %s", name)

	// Reconfigure the GVR
	gvr = schema.GroupVersionResource{
		Group:    "catalog.cattle.io",
		Version:  "v1",
		Resource: "clusterrepos",
	}

	// List of clusterrepos to delete
	names := []string{"rancher-charts", "rancher-rke2-charts", "rancher-partner-charts"}
	for _, name := range names {
		err = dynamicClient.Resource(gvr).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Debugf("Rancher Pre-Upgrade: Failed deleting clusterrepos.catalog.cattle.io %s: %v", name, err)
			return err
		}
		log.Infof("Rancher Pre-Upgrade: Deleted clusterrepos.catalog.cattle.io %s", name)
	}

	return nil
}
