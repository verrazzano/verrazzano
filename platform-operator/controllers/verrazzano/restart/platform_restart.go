// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restart

import (
	"context"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzString "github.com/verrazzano/verrazzano/pkg/string"
	v1 "k8s.io/api/core/v1"
)

type ImageCheckResult int

const (
	OutOfDate ImageCheckResult = iota
	UpToDate
	NotFound
)

// RestartComponents restarts all the deployments, StatefulSets, and DaemonSets
// in all the Istio injected system namespaces
func RestartComponents(log vzlog.VerrazzanoLogger, namespaces []string, generation int64, podMatcher PodMatcher) error {
	if err := podMatcher.ReInit(); err != nil {
		return err
	}
	// Get the go client so we can bypass the cache and get directly from etcd
	goClient, err := k8sutil.GetGoClient(log)
	if err != nil {
		return err
	}

	// Restart all the deployments in the injected system namespaces
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
		podList, err := getPodsForSelector(log, goClient, deployment.Namespace, deployment.Spec.Selector)
		if err != nil {
			return err
		}
		// Check if any pods need the new Istio image injected
		needsNewProxy := podMatcher.Matches(log, podList, "Deployment", deployment.Name)
		if !needsNewProxy {
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
		log.Oncef("Finished restarting deployment %s/%s to pick up new sidecars", deployment.Namespace, deployment.Name)

	}
	// Restart all the StatefulSets in the injected system namespaces
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
		podList, err := getPodsForSelector(log, goClient, sts.Namespace, sts.Spec.Selector)
		if err != nil {
			return err
		}
		// Check if any pods contain the old Istio proxy image
		needsNewProxy := podMatcher.Matches(log, podList, "StatefulSet", sts.Name)
		if !needsNewProxy {
			continue
		}
		if sts.Spec.Template.ObjectMeta.Annotations == nil {
			sts.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		}
		sts.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = buildRestartAnnotationString(generation)
		if _, err := goClient.AppsV1().StatefulSets(sts.Namespace).Update(context.TODO(), sts, metav1.UpdateOptions{}); err != nil {
			return log.ErrorfNewErr("Failed, error updating StatefulSet %s annotation to force a pod restart", sts.Name)
		}
		log.Oncef("Finished restarting statefulSet %s/%s to pick up new sidecars", sts.Namespace, sts.Name)

	}

	// Restart all the DaemonSets in the injected system namespaces
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
		podList, err := getPodsForSelector(log, goClient, daemonSet.Namespace, daemonSet.Spec.Selector)
		if err != nil {
			return err
		}
		// Check if any pods contain the old Istio proxy image
		needsNewProxy := podMatcher.Matches(log, podList, "DaemonSet", daemonSet.Name)
		if !needsNewProxy {
			continue
		}

		if daemonSet.Spec.Template.ObjectMeta.Annotations == nil {
			daemonSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		}
		daemonSet.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = buildRestartAnnotationString(generation)
		if _, err := goClient.AppsV1().DaemonSets(daemonSet.Namespace).Update(context.TODO(), daemonSet, metav1.UpdateOptions{}); err != nil {
			return log.ErrorfNewErr("Failed, error updating DaemonSet %s annotation to force a pod restart", daemonSet.Name)
		}
		log.Oncef("Finished restarting daemonSet %s/%s to pick up new sidecars", daemonSet.Namespace, daemonSet.Name)
	}
	return nil
}

func istioProxyOlderThanTwoMinorVersions(log vzlog.VerrazzanoLogger, imageVersions []string, containerImage, workloadType, workloadName string) bool {
	if strings.Contains(containerImage, proxyv2ImageName) {
		// Container contains the proxy2 image (Envoy Proxy).
		containerImageVersionArray := getImageVersionArray(containerImage)
		istioMinorVersion, _ := strconv.Atoi(imageVersions[1])
		containerImageMinorVersion, _ := strconv.Atoi(containerImageVersionArray[1])
		minorVersionDiff := istioMinorVersion - containerImageMinorVersion

		if strings.Compare(imageVersions[0], containerImageVersionArray[0]) == 1 {
			log.Oncef("%s %s has a pod with an Istio proxy with a major version change%s", workloadType, workloadName, containerImage)
			return true
		} else if minorVersionDiff > 2 {
			log.Oncef("%s %s has a pod with an old Istio proxy where skew is more than 2 minor versions%s", workloadType, workloadName, containerImage)
			return true
		}
	}
	return false
}

// getPodsForSelector gets pods for a namespace matching a selector
func getPodsForSelector(log vzlog.VerrazzanoLogger, client kubernetes.Interface, ns string, labelSelector *metav1.LabelSelector) (*v1.PodList, error) {
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

// isImageOutOfDate returns true if the container image is not as expected (out of date)
func isImageOutOfDate(log vzlog.VerrazzanoLogger, imageName, actualImage, expectedImage, workloadType, workloadName string) ImageCheckResult {
	if strings.Contains(actualImage, imageName) {
		if 0 != strings.Compare(actualImage, expectedImage) {
			return OutOfDate
		}
		return UpToDate
	}
	return NotFound
}

// getImageVersionArray splits a container image string into an array of version numbers from its semantic version
func getImageVersionArray(image string) []string {
	imageSplitArray := strings.SplitN(image, ":", 2)
	return strings.Split(imageSplitArray[1], ".")
}
