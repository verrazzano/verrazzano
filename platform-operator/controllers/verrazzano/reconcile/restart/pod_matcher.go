// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restart

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// PodMatcher implementations returns true/false if a given podList
// match the defined conditions.
type PodMatcher interface {
	ReInit() error
	Matches(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType, workloadName string) bool
}

// WKOPodMatcher matches pods with an out of date Envoy, Fluentd, or WKO Exporter sidecar.
type WKOPodMatcher struct {
	istioProxyImage  string
	wkoExporterImage string
	fluentdImage     string
}

// OutdatedSidecarPodMatcher matches pods with an out of date Envoy or Fluentd sidecar.
type OutdatedSidecarPodMatcher struct {
	fluentdImage    string
	istioProxyImage string
}

// AppPodMatcher matches pods:
// - that are Istio injected without a Envoy sidecar
// - that have an outdated Envoy sidecar
// - that have an outdated Fluentd sidecar
type AppPodMatcher struct {
	fluentdImage    string
	istioProxyImage string
}

// EnvoyOlderThanTwoVersionsPodMatcher matches pods with Envoy images two minor versions or older
type EnvoyOlderThanTwoVersionsPodMatcher struct {
	istioProxyImage string
}

// NoIstioSidecarMatcher matches pods without an Envoy sidecar
type NoIstioSidecarMatcher struct{}

func (o *OutdatedSidecarPodMatcher) ReInit() error {
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
func (o *OutdatedSidecarPodMatcher) Matches(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType, workloadName string) bool {
	for _, pod := range podList.Items {
		for _, c := range pod.Spec.Containers {
			if isImageOutOfDate(log, proxyv2ImageName, c.Image, o.istioProxyImage, workloadType, workloadName) == OutOfDate {
				return true
			}
			if isImageOutOfDate(log, fluentdImageName, c.Image, o.fluentdImage, workloadType, workloadName) == OutOfDate {
				return true
			}
		}
	}
	return false
}

func (w *WKOPodMatcher) ReInit() error {
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
func (w *WKOPodMatcher) Matches(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType, workloadName string) bool {
	istioProxyImageVersionArray := getImageVersionArray(w.istioProxyImage)
	for _, pod := range podList.Items {
		for _, c := range pod.Spec.Containers {
			if isImageOutOfDate(log, fluentdImageName, c.Image, w.fluentdImage, workloadType, workloadName) == OutOfDate {
				return true
			}
			if isImageOutOfDate(log, wkoExporterImageName, c.Image, w.wkoExporterImage, workloadType, workloadName) == OutOfDate {
				return true
			}
			if !istioProxyOlderThanTwoMinorVersions(log, istioProxyImageVersionArray, c.Image, workloadType, workloadName) {
				return true
			}
		}
	}
	return false
}

func (e *EnvoyOlderThanTwoVersionsPodMatcher) ReInit() error {
	images, err := getImages(istioSubcomponent, proxyv2ImageName)
	if err != nil {
		return err
	}
	e.istioProxyImage = images[proxyv2ImageName]
	return nil
}

// Matches when Envoy container is two or more versions out of date
func (e *EnvoyOlderThanTwoVersionsPodMatcher) Matches(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType, workloadName string) bool {
	istioProxyImageVersionArray := getImageVersionArray(e.istioProxyImage)
	for _, pod := range podList.Items {
		for _, container := range pod.Spec.Containers {
			if istioProxyOlderThanTwoMinorVersions(log, istioProxyImageVersionArray, container.Image, workloadType, workloadName) {
				return true
			}
		}
	}
	return false
}

func (a *AppPodMatcher) ReInit() error {
	images, err := getImages(istioSubcomponent, proxyv2ImageName,
		verrazzanoSubcomponent, fluentdImageName)
	if err != nil {
		return err
	}
	a.istioProxyImage = images[proxyv2ImageName]
	a.fluentdImage = images[fluentdImageName]
	return nil
}

// Matches when a pod has an outdated istiod/proxyv2 image, or an outdate fluentd image
func (a *AppPodMatcher) Matches(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType, workloadName string) bool {
	goClient, err := k8sutil.GetGoClient(log)
	if err != nil {
		log.Errorf("Failed to get kubernetes client for AppConfig %s/%s: %v", workloadType, workloadName, err)
		return false
	}
	for _, pod := range podList.Items {
		for _, c := range pod.Spec.Containers {
			if isImageOutOfDate(log, fluentdImageName, c.Image, a.fluentdImage, workloadType, workloadName) == OutOfDate {
				return true
			}
			istioImageStatus := isImageOutOfDate(log, proxyv2ImageName, c.Image, a.istioProxyImage, workloadType, workloadName)
			switch istioImageStatus {
			case OutOfDate:
				return true
			case NotFound:
				podNamespace, _ := goClient.CoreV1().Namespaces().Get(context.TODO(), pod.GetNamespace(), metav1.GetOptions{})
				namespaceLabels := podNamespace.GetLabels()
				value, ok := namespaceLabels["istio-injection"]

				// Ignore OAM pods that do not have Istio injected
				if !ok || value != "enabled" {
					return false
				}
				log.Oncef("Restarting %s %s which has a pod with istio injected namespace", workloadType, workloadName)
				return true
			default:
				return false
			}
		}
	}
	return false
}

func (n *NoIstioSidecarMatcher) ReInit() error {
	return nil
}

// Matches when if any pods don't have an Envoy sidecar
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
