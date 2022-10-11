// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzString "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
)

// RestartCheckFunc is the function used to check if a pod needs to be restarted
type RestartCheckFunc func(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType string, workloadNmae string, istioProxyImageName string) bool

// RestartComponents restarts all the deployments, StatefulSets, and DaemonSets
// in all of the Istio injected system namespaces
func RestartComponents(log vzlog.VerrazzanoLogger, namespaces []string, generation int64, restartCheckFunc RestartCheckFunc) error {
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
	log.Oncef("Restarting system Deployments to pickup latest Istio proxy sidecar")
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
		log.Oncef("Checking the Istio proxy sidecar for Deployment %s", deployment.Name)

		// Get the pods for this deployment
		podList, err := getMatchingPods(log, goClient, deployment.Namespace, deployment.Spec.Selector)
		if err != nil {
			return err
		}
		// Check if any pods contain the old Istio proxy image
		found := restartCheckFunc(log, podList, "Deployment", deployment.Name, istioProxyImage)
		if !found {
			continue
		}
		// Annotate the deployment to do a restart of the pods
		if deployment.Spec.Paused {
			return log.ErrorfNewErr("Failed, Deployment %s can't be restarted because it is paused", deployment.Name)
		}
		if deployment.Spec.Template.ObjectMeta.Annotations == nil {
			deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		}
		deployment.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = buildRestartAnnotationString(generation)
		if _, err := goClient.AppsV1().Deployments(deployment.Namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{}); err != nil {
			return log.ErrorfNewErr("Failed, error updating Deployment %s annotation to force a pod restart", deployment.Name)
		}
	}
	log.Oncef("Finished restarting system Deployments to pick up the latest Istio proxy sidecar")

	// Restart all the StatefulSets in the injected system namespaces
	log.Oncef("Restarting system StatefulSets to pickup latest Istio proxy sidecar")
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
		log.Oncef("Checking the Istio proxy sidecar for StatefulSet %s", sts.Name)

		// Get the pods for this StatefulSet
		podList, err := getMatchingPods(log, goClient, sts.Namespace, sts.Spec.Selector)
		if err != nil {
			return err
		}
		// Check if any pods contain the old Istio proxy image
		found := restartCheckFunc(log, podList, "StatefulSet", sts.Name, istioProxyImage)
		if !found {
			continue
		}
		if sts.Spec.Template.ObjectMeta.Annotations == nil {
			sts.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		}
		sts.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = buildRestartAnnotationString(generation)
		if _, err := goClient.AppsV1().StatefulSets(sts.Namespace).Update(context.TODO(), sts, metav1.UpdateOptions{}); err != nil {
			return log.ErrorfNewErr("Failed, error updating StatefulSet %s annotation to force a pod restart", sts.Name)
		}
	}
	log.Oncef("Finished restarting system Statefulsets to pick up latest Istio proxy sidecar")

	// Restart all the DaemonSets in the injected system namespaces
	log.Oncef("Restarting system DaemonSets to pickup latest Istio proxy sidecar")
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
		log.Oncef("Checking the Istio proxy sidecar for DaemonSet %s", daemonSet.Name)

		// Get the pods for this DaemonSet
		podList, err := getMatchingPods(log, goClient, daemonSet.Namespace, daemonSet.Spec.Selector)
		if err != nil {
			return err
		}
		// Check if any pods contain the old Istio proxy image
		found := restartCheckFunc(log, podList, "DaemonSet", daemonSet.Name, istioProxyImage)
		if !found {
			continue
		}

		if daemonSet.Spec.Template.ObjectMeta.Annotations == nil {
			daemonSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		}
		daemonSet.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = buildRestartAnnotationString(generation)
		if _, err := goClient.AppsV1().DaemonSets(daemonSet.Namespace).Update(context.TODO(), daemonSet, metav1.UpdateOptions{}); err != nil {
			return log.ErrorfNewErr("Failed, error updating DaemonSet %s annotation to force a pod restart", daemonSet.Name)
		}
	}
	log.Oncef("Finished restarting system DaemonSets to pick up latest Istio proxy sidecar")
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

// DoesPodContainOldIstioSidecar returns true if any pods contain an old Istio proxy sidecar
func DoesPodContainOldIstioSidecar(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType string, workloadName string, istioProxyImageName string) bool {
	for _, pod := range podList.Items {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "proxyv2") {
				// Container contains the proxy2 image (Envoy Proxy).  Return true if it
				// doesn't match the Istio proxy in the BOM
				if 0 != strings.Compare(container.Image, istioProxyImageName) {
					log.Oncef("Restarting %s %s which has a pod with an old Istio proxy %s", workloadType, workloadName, container.Image)
					return true
				}
			}
		}
	}
	return false
}

func DoesPodContainOldIstioSidecarSkew2MinorVersion(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType string, workloadName string, istioProxyImageName string) bool {
	istoiProxyImageSplitArray := strings.SplitN(istioProxyImageName, ":", 2)
	istioProxyImageVersionArray := strings.Split(istoiProxyImageSplitArray[1], ".")
	fmt.Println("ISTIO PROXY IMAGE ARRAY:", istioProxyImageVersionArray)
	for _, pod := range podList.Items {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "proxyv2") {
				// Container contains the proxy2 image (Envoy Proxy).  Return true if it
				// doesn't match the Istio proxy in the BOM
				containerImageSplit := strings.SplitN(container.Image, ":", 2)
				containerImageVersionArray := strings.Split(containerImageSplit[1], ".")
				fmt.Println("CONTAINER IMAGE ARRAY:", containerImageVersionArray)

				if strings.Compare(istioProxyImageVersionArray[0], containerImageVersionArray[0]) == 1 {
					log.Oncef("Restarting %s %s which has a pod with an old Istio proxy with skew of more than 2 minor versions%s", workloadType, workloadName, container.Image)
					return true
				} else if strings.Compare(istioProxyImageVersionArray[1], containerImageVersionArray[1]) >= 0 {
					log.Oncef("Restarting %s %s which has a pod with an old Istio proxy with skew of more than 2 minor versions%s", workloadType, workloadName, container.Image)
					return true
				}
			}
		}
	}
	return false
}

// DoesPodContainNoIstioSidecar returns true if any pods don't have an Istio proxy sidecar
func DoesPodContainNoIstioSidecar(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType string, workloadName string, _ string) bool {
	for _, pod := range podList.Items {
		// Ignore pods that are not expected to have Istio injected
		noInjection := false
		for _, item := range config.GetNoInjectionComponents() {
			if strings.Contains(pod.Name, item) {
				noInjection = true
				break
			}
		}
		if noInjection {
			continue
		}
		proxyFound := false
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "proxyv2") {
				proxyFound = true
			}
		}
		if !proxyFound {
			log.Oncef("Restarting %s %s which has a pod with no Istio proxy image", workloadType, workloadName)
			return true
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
func buildRestartAnnotationString(generation int64) string {
	return strconv.Itoa(int(generation))
}
