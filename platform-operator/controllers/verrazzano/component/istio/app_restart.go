// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"strconv"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	oam "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RestartApps restarts all the applications that have old Istio sidecars.
// It also restarts WebLogic domains that were stopped in Istio pre-upgrade
func RestartApps(log vzlog.VerrazzanoLogger, client clipkg.Client, generation int64) error {
	// Generate a restart version that will not change for this Verrazzano version
	restartVersion := "upgrade-" + strconv.Itoa(int(generation))

	// Start WebLogic domains that were shutdown
	log.Infof("Starting WebLogic domains that were stopped pre-upgrade")
	if err := startDomainsStoppedByUpgrade(log, client, restartVersion); err != nil {
		return err
	}

	// Rolling restart Weblogic domain pods if the Istio version skew is 2 minor versions max
	if err := RestartDomainsUsingOldEnvoyMaxSkewTwoMinorVersions(log, client, restartVersion); err != nil {
		return err
	}

	// Restart all other apps
	log.Infof("Restarting all applications so they can get the new Envoy sidecar")
	if err := restartAllApps(log, client, restartVersion); err != nil {
		return err
	}
	return nil
}

// StopDomainsUsingOldEnvoy stops all the WebLogic domains using Envoy 1.7.3
func StopDomainsUsingOldEnvoy(log vzlog.VerrazzanoLogger, client clipkg.Client) error {
	// Get the latest Istio proxy image name from the bom
	istioProxyImage, err := getIstioProxyImageFromBom()
	if err != nil {
		return log.ErrorfNewErr("Failed, restart components cannot find Istio proxy image in BOM: %v", err)
	}

	// get all the app configs
	appConfigs := oam.ApplicationConfigurationList{}
	if err := client.List(context.TODO(), &appConfigs, &clipkg.ListOptions{}); err != nil {
		return log.ErrorfNewErr("Failed to list appConfigs %v", err)
	}

	// Loop through the WebLogic workloads and stop the ones that need to be stopped
	for _, appConfig := range appConfigs.Items {
		log.Debugf("StopWebLogicApps: found appConfig %s", appConfig.Name)
		for _, wl := range appConfig.Status.Workloads {
			if wl.Reference.Kind == vzconst.VerrazzanoWebLogicWorkloadKind {
				if err := stopDomainIfNeeded(log, client, appConfig, wl.Reference.Name, istioProxyImage); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// RestartDomainsUsingOldEnvoyMaxSkewTwoMinorVersions restarts all the WebLogic domains using Envoy 1.7.3
func RestartDomainsUsingOldEnvoyMaxSkewTwoMinorVersions(log vzlog.VerrazzanoLogger, client clipkg.Client, restartVersion string) error {
	log.Infof("RestartDomainsUsingOldEnvoyMaxSkewTwoMinorVersions -------")
	// Generate a restart version that will not change for this Verrazzano version
	//restartVersion := "upgrade-" + strconv.Itoa(int(generation))

	// Get the latest Istio proxy image name from the bom
	istioProxyImage, err := getIstioProxyImageFromBom()
	if err != nil {
		return log.ErrorfNewErr("Failed, restart components cannot find Istio proxy image in BOM: %v", err)
	}

	// get all the app configs
	appConfigs := oam.ApplicationConfigurationList{}
	if err := client.List(context.TODO(), &appConfigs, &clipkg.ListOptions{}); err != nil {
		return log.ErrorfNewErr("Failed to list appConfigs %v", err)
	}

	// Loop through the WebLogic workloads and restarts the ones that need to be restarted
	for _, appConfig := range appConfigs.Items {
		log.Debugf("RestartWebLogicApps: found appConfig %s", appConfig.Name)
		for _, wl := range appConfig.Status.Workloads {
			if wl.Reference.Kind == vzconst.VerrazzanoWebLogicWorkloadKind {
				log.Infof("Before restartDomainIfNeeded -------")
				if err := restartDomainIfNeeded(log, client, appConfig, wl.Reference.Name, istioProxyImage, restartVersion); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Determine if the WebLogic domain needs to be stopped, if so then stop it
func stopDomainIfNeeded(log vzlog.VerrazzanoLogger, client clipkg.Client, appConfig oam.ApplicationConfiguration, wlName string, istioProxyImage string) error {
	log.Progressf("StopWebLogicApps: checking if domain for workload %s needs to be stopped", wlName)

	// Get the go client so we can bypass the cache and get directly from etcd
	goClient, err := k8sutil.GetGoClient(log)
	if err != nil {
		return err
	}

	// Get the domain pods for this workload
	weblogicReq, _ := labels.NewRequirement("verrazzano.io/workload-type", selection.Equals, []string{"weblogic"})
	compReq, _ := labels.NewRequirement("app.oam.dev/component", selection.Equals, []string{wlName})
	appConfNameReq, _ := labels.NewRequirement("app.oam.dev/name", selection.Equals, []string{appConfig.Name})
	selector := labels.NewSelector()
	selector = selector.Add(*weblogicReq).Add(*compReq).Add(*appConfNameReq)

	// Get the pods using the label selector
	podList, err := goClient.CoreV1().Pods(appConfig.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return log.ErrorfNewErr("Failed to list pods for Domain %s/%s: %v", appConfig.Namespace, wlName, err)
	}

	// Check if any pods contain the old Istio proxy image
	found := DoesPodContainOldIstioSidecarSkewGreaterThanTwoMinorVersion(log, podList, "OAM WebLogic Domain", wlName, istioProxyImage)
	if !found {
		return nil
	}

	return stopDomain(client, appConfig.Namespace, wlName)
}

// Determine if the WebLogic domain needs to be restarted
func restartDomainIfNeeded(log vzlog.VerrazzanoLogger, client clipkg.Client, appConfig oam.ApplicationConfiguration, wlName string, istioProxyImage string, restartVersion string) error {
	log.Progressf("RestartWebLogicApps: checking if domain for workload %s needs to be restarted", wlName)

	// Get the go client so we can bypass the cache and get directly from etcd
	goClient, err := k8sutil.GetGoClient(log)
	if err != nil {
		return err
	}

	// Get the domain pods for this workload
	weblogicReq, _ := labels.NewRequirement("verrazzano.io/workload-type", selection.Equals, []string{"weblogic"})
	compReq, _ := labels.NewRequirement("app.oam.dev/component", selection.Equals, []string{wlName})
	appConfNameReq, _ := labels.NewRequirement("app.oam.dev/name", selection.Equals, []string{appConfig.Name})
	selector := labels.NewSelector()
	selector = selector.Add(*weblogicReq).Add(*compReq).Add(*appConfNameReq)

	// Get the pods using the label selector
	podList, err := goClient.CoreV1().Pods(appConfig.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return log.ErrorfNewErr("Failed to list pods for Domain %s/%s: %v", appConfig.Namespace, wlName, err)
	}

	// Check if weblogic domain pods contain the old Istio proxy image where skew is 2 minor versions at max.
	found := !DoesPodContainOldIstioSidecarSkewGreaterThanTwoMinorVersion(log, podList, "OAM WebLogic Domain", wlName, istioProxyImage)
	if !found {
		return nil
	}

	return restartDomain(log, client, appConfig.Namespace, wlName, restartVersion)
}

// Stop the WebLogic domain
func stopDomain(client clipkg.Client, wlNamespace string, wlName string) error {
	// Set the lifecycle annotation on the VerrazzanoWebLogicWorkload
	var wl vzapp.VerrazzanoWebLogicWorkload
	wl.Namespace = wlNamespace
	wl.Name = wlName
	_, err := controllerutil.CreateOrUpdate(context.TODO(), client, &wl, func() error {
		if wl.ObjectMeta.Annotations == nil {
			wl.ObjectMeta.Annotations = make(map[string]string)
		}
		wl.ObjectMeta.Annotations[vzconst.LifecycleActionAnnotation] = vzconst.LifecycleActionStop
		return nil
	})
	return err
}

// Restart the WebLogic domain
func restartDomain(log vzlog.VerrazzanoLogger, client clipkg.Client, wlNamespace string, wlName string, restartVersion string) error {
	log.Infof("restartDomain -------")
	var wl vzapp.VerrazzanoWebLogicWorkload
	wl.Namespace = wlNamespace
	wl.Name = wlName
	_, err := controllerutil.CreateOrUpdate(context.TODO(), client, &wl, func() error {
		if wl.ObjectMeta.Annotations == nil {
			wl.ObjectMeta.Annotations = make(map[string]string)
		}
		log.Infof("restartDomain--------Update restart Version  -------")
		wl.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = restartVersion
		return nil
	})
	return err
}

// startDomainsStoppedByUpgrade starts all the WebLogic domains that upgrade previously stopped
func startDomainsStoppedByUpgrade(log vzlog.VerrazzanoLogger, client clipkg.Client, restartVersion string) error {
	log.Progressf("RestartApps: checking if any domains need to be started")

	// get all the app configs
	appConfigs := oam.ApplicationConfigurationList{}
	if err := client.List(context.TODO(), &appConfigs, &clipkg.ListOptions{}); err != nil {
		return log.ErrorfNewErr("Failed to list appConfigs %v", err)
	}

	// Loop through the WebLogic workloads and start the ones that were stopped
	for _, appConfig := range appConfigs.Items {
		log.Debugf("RestartApps: found appConfig %s", appConfig.Name)
		for _, wl := range appConfig.Status.Workloads {
			if wl.Reference.Kind == vzconst.VerrazzanoWebLogicWorkloadKind {
				if err := startDomainIfNeeded(log, client, appConfig.Namespace, wl.Reference.Name, restartVersion); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Start the WebLogic domain if upgrade stopped it
func startDomainIfNeeded(log vzlog.VerrazzanoLogger, client clipkg.Client, wlNamespace string, wlName string, restartVersion string) error {
	// Set the lifecycle annotation on the VerrazzanoWebLogicWorkload
	var wl vzapp.VerrazzanoWebLogicWorkload
	wl.Namespace = wlNamespace
	wl.Name = wlName
	_, err := controllerutil.CreateOrUpdate(context.TODO(), client, &wl, func() error {
		if wl.ObjectMeta.Annotations == nil {
			return nil
		}
		if wl.ObjectMeta.Annotations[vzconst.LifecycleActionAnnotation] == vzconst.LifecycleActionStop {
			log.Debugf("Workload %s lifecycle annotation is 'stop',  Changing it to 'start'", wlName)
			wl.ObjectMeta.Annotations[vzconst.LifecycleActionAnnotation] = vzconst.LifecycleActionStart
		}
		// Set the restart version also so that when the app config is modified to use that
		// restart version, it will the same version so WebLogic will not start twice
		log.Debugf("RestartApps: setting restart version for workload %s to %s ...  Old version is %s", wlName,
			restartVersion, wl.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation])
		wl.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = restartVersion
		return nil
	})
	return err
}

// restartAllApps restarts all the applications
func restartAllApps(log vzlog.VerrazzanoLogger, client clipkg.Client, restartVersion string) error {
	log.Progressf("Restarting all OAM applications that have an old Istio proxy sidecar")

	// Get the latest Istio proxy image name from the bom
	istioProxyImage, err := getIstioProxyImageFromBom()
	if err != nil {
		return log.ErrorfNewErr("Failed, restart components cannot find Istio proxy image in BOM: %v", err)
	}

	// get the go client so we can bypass the cache and get directly from etcd
	goClient, err := k8sutil.GetGoClient(log)
	if err != nil {
		return err
	}

	// get all the app configs
	appConfigs := oam.ApplicationConfigurationList{}
	if err := client.List(context.TODO(), &appConfigs, &clipkg.ListOptions{}); err != nil {
		return log.ErrorfNewErr("Failed to listing appConfigs %v", err)
	}

	// check each app config to see if any of the pods have old Istio proxy images
	for _, appConfig := range appConfigs.Items {
		log.Oncef("Checking OAM Application %s pods for an old Istio proxy sidecar", appConfig.Name)

		// Get the pods for this appconfig
		appConfNameReq, _ := labels.NewRequirement("app.oam.dev/name", selection.Equals, []string{appConfig.Name})
		weblogicReq, _ := labels.NewRequirement("verrazzano.io/workload-type", selection.NotEquals, []string{"weblogic"})
		selector := labels.NewSelector()
		selector = selector.Add(*appConfNameReq).Add(*weblogicReq)

		// Get the pods using the label selector
		podList, err := goClient.CoreV1().Pods(appConfig.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return log.ErrorfNewErr("Failed to list pods for AppConfig %s/%s: %v", appConfig.Namespace, appConfig.Name, err)
		}

		//Check if any pods that contain no or old istio proxy container with istio injection labeled namespace
		foundOAMPodRequireRestart, _ := DoesAppPodNeedRestart(log, podList, "OAM Application", appConfig.Name, istioProxyImage)

		if foundOAMPodRequireRestart {
			err := restartOAMApp(log, appConfig, client, restartVersion)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// DoesAppPodNeedRestart returns true if any OAM pods with istio injected don't have or have an old Istio proxy sidecar
func DoesAppPodNeedRestart(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType string, workloadName string, istioProxyImageName string) (bool, error) {
	// Return true if the pod has an old Istio proxy container
	for _, pod := range podList.Items {
		doesProxyExists := false
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "proxyv2") {
				doesProxyExists = true
				// Container contains the proxy2 image (Envoy Proxy).  Return true if it
				// doesn't match the Istio proxy in the BOM
				if 0 != strings.Compare(container.Image, istioProxyImageName) {
					log.Oncef("Restarting %s %s which has a pod with an old Istio proxy %s", workloadType, workloadName, container.Image)
					return true, nil
				}
				break
			}
		}
		if doesProxyExists {
			return false, nil
		}
		goClient, err := k8sutil.GetGoClient(log)
		if err != nil {
			return false, log.ErrorfNewErr("Failed to get kubernetes client for AppConfig %s/%s: %v", workloadType, workloadName, err)
		}
		podNamespace, _ := goClient.CoreV1().Namespaces().Get(context.TODO(), pod.GetNamespace(), metav1.GetOptions{})
		namespaceLabels := podNamespace.GetLabels()
		value, ok := namespaceLabels["istio-injection"]

		// Ignore OAM pods that do not have Istio injected
		if !ok || value != "enabled" {
			continue
		}
		log.Oncef("Restarting %s %s which has a pod with istio injected namespace", workloadType, workloadName)
		return true, nil
	}
	return false, nil
}

// restartOAMApp sets the restart version for appconfig to recycle the pod
func restartOAMApp(log vzlog.VerrazzanoLogger, appConfig oam.ApplicationConfiguration, client clipkg.Client, restartVersion string) error {

	// Set the update restart version
	var ac oam.ApplicationConfiguration
	ac.Namespace = appConfig.Namespace
	ac.Name = appConfig.Name
	_, err := controllerutil.CreateOrUpdate(context.TODO(), client, &ac, func() error {
		if ac.ObjectMeta.Annotations == nil {
			ac.ObjectMeta.Annotations = make(map[string]string)
		}
		log.Progressf("Setting restart version for appconfig %s to %s. Previous version is %s", appConfig.Name,
			restartVersion, ac.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation])
		ac.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = restartVersion
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
