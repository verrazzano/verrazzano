// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"errors"
	"strconv"
	"strings"

	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzString "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// RestartComponents restarts all the deployments, StatefulSets, and DaemonSets
// in all of the Istio injected system namespaces
func RestartComponents(log vzlog.VerrazzanoLogger, namespaces []string, client clipkg.Client, cr *installv1alpha1.Verrazzano) error {
	// Get the latest Istio proxy image name from the bom
	istioProxyImage, err := getIstioProxyImageFromBom()
	if err != nil {
		return log.ErrorfNewErr("Restart components cannot find Istio proxy image in BOM: %v", err)
	}
	// Get the go client so we can bypass the cache and get directly from etcd
	goClient, err := k8sutil.GetGoClient(log)
	if err != nil {
		return err
	}

	// Restart all the deployments in the injected system namespaces
	log.Oncef("Restarting system Deployments that have an old Istio sidecar so that the pods get the new Isio sidecar")
	deploymentList, err := goClient.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for index := range deploymentList.Items {
		deployment := &deploymentList.Items[index]

		// Ignore deployment if it is NOT in an Istio injected system namespace
		if !vzString.SliceContainsString(namespaces, deployment.Namespace) {
			continue
		}
		// Get the pods for this deployment
		podList, err := getMatchingPods(log, goClient, deployment.Namespace, deployment.Spec.Selector)
		if err != nil {
			return err
		}
		// Check if any pods contain the old Istio proxy image
		if !doesPodContainOldIstioSidecar(podList, istioProxyImage) {
			continue
		}
		// Annotate the deployment to do a restart of the pods
		if deployment.Spec.Paused {
			return log.ErrorfNewErr("Failed, deployment %s can't be restarted because it is paused", deployment.Name)
		}
		if deployment.Spec.Template.ObjectMeta.Annotations == nil {
			deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		}
		deployment.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = buildRestartAnnotationString(cr)
		if err := client.Update(context.TODO(), deployment); err != nil {
			return log.ErrorfNewErr("Failed, error updating deployment %s annotation to force a pod restart", deployment.Name)
		}
		log.Oncef("Updated deployment %s annotation causing a pod restart to get new Istio sidecar", deployment.Name)
	}
	log.Oncef("Finished restarting system Deployments in istio injected namespaces to pick up new Isio sidecar")

	// Restart all the StatefulSets in the injected system namespaces
	log.Oncef("Restarting system StatefulSets that have an old Istio sidecar so that the pods get the new Isio sidecar")
	statefulSetList, err := goClient.AppsV1().StatefulSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for index := range statefulSetList.Items {
		sts := &statefulSetList.Items[index]

		// Ignore StatefulSet if it is NOT in an Istio injected system namespace
		if !vzString.SliceContainsString(namespaces, sts.Namespace) {
			continue
		}
		// Get the pods for this StatefulSet
		podList, err := getMatchingPods(log, goClient, sts.Namespace, sts.Spec.Selector)
		if err != nil {
			return err
		}
		// Check if any pods contain the old Istio proxy image
		if !doesPodContainOldIstioSidecar(podList, istioProxyImage) {
			continue
		}
		if sts.Spec.Template.ObjectMeta.Annotations == nil {
			sts.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		}
		sts.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = buildRestartAnnotationString(cr)
		if err := client.Update(context.TODO(), sts); err != nil {
			return log.ErrorfNewErr("Failed, error updating StatefulSet %s annotation to force a pod restart", sts.Name)
		}
		log.Oncef("Updated StatefulSet %s annotation causing a pod restart to get new Istio sidecar", sts.Name)
	}
	log.Oncef("Restarted system Statefulsets in istio injected namespaces")

	// Restart all the DaemonSets in the injected system namespaces
	log.Oncef("Restarting system DaemonSets that have an old Istio sidecar so that the pods get the new Isio sidecar")
	daemonSetList, err := goClient.AppsV1().DaemonSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for index := range daemonSetList.Items {
		daemonSet := &daemonSetList.Items[index]

		// Ignore StatefulSet if it is NOT in an Istio injected system namespace
		if !vzString.SliceContainsString(namespaces, daemonSet.Namespace) {
			continue
		}
		// Get the pods for this DaemonSet
		podList, err := getMatchingPods(log, goClient, daemonSet.Namespace, daemonSet.Spec.Selector)
		if err != nil {
			return err
		}
		// Check if any pods contain the old Istio proxy image
		if !doesPodContainOldIstioSidecar(podList, istioProxyImage) {
			continue
		}
		if daemonSet.Spec.Template.ObjectMeta.Annotations == nil {
			daemonSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		}
		daemonSet.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = buildRestartAnnotationString(cr)
		if err := client.Update(context.TODO(), daemonSet); err != nil {
			return log.ErrorfNewErr("Failed, error updating DaemonSet %s annotation to force a pod restart", daemonSet.Name)
		}
		log.Oncef("Updated DaemonSet %s annotation causing a pod restart to get new Istio sidecar", daemonSet.Name)
	}
	log.Oncef("Restarted system DaemonSets in istio injected namespaces")
	return nil
}

// Get the Istio proxy image from the Istiod subcomponent in the BOM
func getIstioProxyImageFromBom() (string, error) {
	// Create a Bom and get the Key Value overrides
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return "", errors.New("Failed to get access to the BOM")
	}
	images, err := bomFile.GetImageNameList(subcompIstiod)
	if err != nil {
		return "", errors.New("Failed to get the images for Istiod")
	}
	for i, image := range images {
		if strings.Contains(image, "proxyv2") {
			return images[i], nil
		}
	}
	return "", errors.New("Failed to find Istio proxy image in the BOM for Istiod")
}

// doesPodContainOldIstioSidecar returns true if any pods contain an old Istio proxy sidecar
func doesPodContainOldIstioSidecar(podList *v1.PodList, istioProxyImageName string) bool {
	for _, pod := range podList.Items {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "proxyv2") {
				// Container contains the proxy2 image (Envoy Proxy).  Return true if it
				// doesn't match the Istio proxy in the BOM
				if 0 != strings.Compare(container.Image, istioProxyImageName) {
					return true
				}
			}
		}
	}
	return false
}

// Get the matching pods in namespace given a selector
func getMatchingPods(log vzlog.VerrazzanoLogger, client kubernetes.Interface, ns string, labelSelector *metav1.LabelSelector) (*v1.PodList, error) {
	// Conver the resource labelselector to a go-client label selector
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, log.ErrorfNewErr("Failed converting metav1.LabelSelector to labels.Selector: %v", err)
	}

	podList, err := client.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, log.ErrorfNewErr("Failed listing pods by label selector: %v", err)
	}
	return podList, nil
}

// Use the CR generation so that we only restart the workloads once
func buildRestartAnnotationString(cr *installv1alpha1.Verrazzano) string {
	return strconv.Itoa(int(cr.Generation))
}
