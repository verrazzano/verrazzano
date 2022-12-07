// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restart

import (
	"context"
	"errors"
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

type PodMatcher interface {
	ReInit() error
	Matches(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType, workloadName string) bool
}

// RestartCheckFunc is the function used to check if a pod needs to be restarted
type RestartCheckFunc func(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType string, workloadNmae string, istioProxyImageName string) bool

// RestartComponents restarts all the deployments, StatefulSets, and DaemonSets
// in all of the Istio injected system namespaces
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
		log.Oncef("Checking sidecars for Deployment %s", deployment.Name)

		// Get the pods for this deployment
		podList, err := getMatchingPods(log, goClient, deployment.Namespace, deployment.Spec.Selector)
		if err != nil {
			return err
		}
		// Check if any pods contain the old Istio proxy image
		found := podMatcher.Matches(log, podList, "Deployment", deployment.Name)
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
	log.Oncef("Finished restarting system Deployments to pick up new sidecars")

	// Restart all the StatefulSets in the injected system namespaces
	log.Oncef("Restarting system StatefulSets to pickup latest sidecars")
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
		log.Oncef("Checking sidecars for StatefulSet %s", sts.Name)

		// Get the pods for this StatefulSet
		podList, err := getMatchingPods(log, goClient, sts.Namespace, sts.Spec.Selector)
		if err != nil {
			return err
		}
		// Check if any pods contain the old Istio proxy image
		found := podMatcher.Matches(log, podList, "StatefulSet", sts.Name)
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
	log.Oncef("Finished restarting system Statefulsets to pick up latest sidecars")

	// Restart all the DaemonSets in the injected system namespaces
	log.Oncef("Restarting system DaemonSets to pickup latest sidecars")
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
		log.Oncef("Checking sidecars for DaemonSet %s", daemonSet.Name)

		// Get the pods for this DaemonSet
		podList, err := getMatchingPods(log, goClient, daemonSet.Namespace, daemonSet.Spec.Selector)
		if err != nil {
			return err
		}
		// Check if any pods contain the old Istio proxy image
		found := podMatcher.Matches(log, podList, "DaemonSet", daemonSet.Name)
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
	log.Oncef("Finished restarting system DaemonSets to pick up latest sidecars")
	return nil
}

// Get the Istio proxy image from the Istiod subcomponent in the BOM
func getIstioProxyImageFromBom() (string, error) {
	// Create a Bom and get the Key Value overrides
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return "", errors.New("Failed to get access to the BOM")
	}
	images, err := bomFile.GetImageNameList("istiod")
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

// OutdatedSidecarMatcher checks if istiod/proxyv2 or fluentd sidecar images are out of date
type OutdatedSidecarMatcher struct {
	fluentdImage    string
	istioProxyImage string
}

func (o *OutdatedSidecarMatcher) ReInit() error {
	images, err := getImages(istioSubcomponent, proxyv2ImageName,
		verrazzanoSubcomponent, fluentdImageName)
	if err != nil {
		return err
	}
	o.istioProxyImage = images[proxyv2ImageName]
	o.fluentdImage = images[fluentdImageName]
	return nil
}

// Matches when a pod has an outdated istiod/proxyv2 image, or an outdate fluentd image
func (o *OutdatedSidecarMatcher) Matches(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType, workloadName string) bool {
	for _, pod := range podList.Items {
		for _, c := range pod.Spec.Containers {
			if matchesImageAndOutOfDate(log, proxyv2ImageName, c.Image, o.istioProxyImage, workloadType, workloadName) {
				return true
			}
			if matchesImageAndOutOfDate(log, fluentdImageName, c.Image, o.fluentdImage, workloadType, workloadName) {
				return true
			}
		}
	}
	return false
}

func matchesImageAndOutOfDate(log vzlog.VerrazzanoLogger, imageName, containerImage, expectedImage, workloadType, workloadName string) bool {
	if strings.Contains(containerImage, imageName) {
		if 0 != strings.Compare(containerImage, expectedImage) {
			log.Oncef("Restarting %s %s which has a pod with an old Istio proxy %s", workloadType, workloadName, containerImage)
			return true
		}
	}
	return false
}

type WKOMatcher struct {
	istioProxyImage  string
	wkoExporterImage string
	fluentdImage     string
}

func (w *WKOMatcher) ReInit() error {
	images, err := getImages(istioSubcomponent, proxyv2ImageName,
		verrazzanoSubcomponent, fluentdImageName,
		wkoSubcomponent, wkoExporterImageName)
	if err != nil {
		return err
	}
	w.istioProxyImage = images[proxyv2ImageName]
	w.fluentdImage = images[fluentdImageName]
	w.wkoExporterImage = images[wkoExporterImageName]
	return nil
}

// Matches when the pod has out of date fluentd, wko exporter, or istio envoy sidecars. The envoy sidecars must be 2 or more
// minor versions out of date.
func (w *WKOMatcher) Matches(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType, workloadName string) bool {
	istioProxyImageVersionArray := getImageVersionArray(w.istioProxyImage)
	for _, pod := range podList.Items {
		for _, c := range pod.Spec.Containers {
			if matchesImageAndOutOfDate(log, fluentdImageName, c.Image, w.fluentdImage, workloadType, workloadName) {
				return true
			}
			if matchesImageAndOutOfDate(log, wkoExporterImageName, c.Image, w.wkoExporterImage, workloadType, workloadName) {
				return true
			}
			if istioProxyOlderThanTwoMinorVersions(log, istioProxyImageVersionArray, c.Image, workloadType, workloadName) {
				return true
			}
		}
	}
	return false
}

func getImageVersionArray(image string) []string {
	imageSplitArray := strings.SplitN(image, ":", 2)
	return strings.Split(imageSplitArray[1], ".")
}

func istioProxyOlderThanTwoMinorVersions(log vzlog.VerrazzanoLogger, imageVersions []string, containerImage, workloadType, workloadName string) bool {
	if strings.Contains(containerImage, proxyv2ImageName) {
		// Container contains the proxy2 image (Envoy Proxy).

		containerImageSplit := strings.SplitN(containerImage, ":", 2)
		containerImageVersionArray := strings.Split(containerImageSplit[1], ".")
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

// DoesPodContainOldIstioSidecarSkewGreaterThanTwoMinorVersion returns true if weblogic domain pods contain old istio sidecar where skew is more than 2 minor versions/skew is 1 major version
func DoesPodContainOldIstioSidecarSkewGreaterThanTwoMinorVersion(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType string, workloadName string, istioProxyImageName string) bool {
	istioProxyImageVersionArray := getImageVersionArray(istioProxyImageName)
	for _, pod := range podList.Items {
		for _, container := range pod.Spec.Containers {
			if istioProxyOlderThanTwoMinorVersions(log, istioProxyImageVersionArray, container.Image, workloadType, workloadName) {
				return true
			}
		}
	}
	return false
}

type NoIstioSidecarMatcher struct{}

func (n *NoIstioSidecarMatcher) ReInit() error {
	return nil
}

// Matches when if any pods don't have an Istio proxy sidecar
func (n *NoIstioSidecarMatcher) Matches(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType, workloadName string) bool {
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
			if strings.Contains(container.Image, proxyv2ImageName) {
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
