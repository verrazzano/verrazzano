// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"strconv"

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
	found := DoesPodContainOldIstioSidecar(log, podList, "OAM WebLogic Domain", wlName, istioProxyImage)
	if !found {
		return nil
	}

	return stopDomain(client, appConfig.Namespace, wlName)
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
		selector := labels.NewSelector()
		selector = selector.Add(*appConfNameReq)

		// Get the pods using the label selector
		podList, err := goClient.CoreV1().Pods(appConfig.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return log.ErrorfNewErr("Failed to list pods for AppConfig %s/%s: %v", appConfig.Namespace, appConfig.Name, err)
		}

		// Check if any pods contain the old Istio proxy image
		foundOldIstioProxyImage := DoesPodContainOldIstioSidecar(log, podList, "OAM Application", appConfig.Name, istioProxyImage)

		//Check if any pods that contain no istio proxy container with istio injection labeled namespace
		foundOAMPodWithoutIstioProxy, _ := DoesOAMPodsContainNoIstioSidecar(log, podList, "OAM Application", appConfig.Name, istioProxyImage)

		if foundOldIstioProxyImage || foundOAMPodWithoutIstioProxy {

			restartOAMApp(log, appConfig, client, restartVersion)
		}
	}
	return nil
}

func restartOAMApp(log vzlog.VerrazzanoLogger, appConfig oam.ApplicationConfiguration, client clipkg.Client, restartVersion string) error {

	// Set the update the restart version
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
