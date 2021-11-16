// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	oam "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

// StopDomainsUsingOldEnvoy stops all the WebLogic domains using Envoy 1.7.3
func StopDomainsUsingOldEnvoy(log *zap.SugaredLogger, client clipkg.Client) error {
	// get all the app configs
	appConfigs := oam.ApplicationConfigurationList{}
	if err := client.List(context.TODO(), &appConfigs, &clipkg.ListOptions{}); err != nil {
		log.Errorf("Error Listing appConfigs %v", err)
		return err
	}

	// Loop through the WebLogic workloads and stop the ones that need to be stopped
	for _, appConfig := range appConfigs.Items {
		for _, wl := range appConfig.Status.Workloads {
			if wl.Reference.Kind == vzconst.VerrazzanoWebLogicWorkloadKind {
				if err := stopDomainIfNeeded(log, client, &appConfig, wl.Reference.Name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Determine if the WebLogic operator needs to be stopped, if so then stop it
func stopDomainIfNeeded(log *zap.SugaredLogger, client clipkg.Client, appConfig *oam.ApplicationConfiguration, wlName string) error {

	// Get the domain pods for this workload
	weblogicReq, _ := labels.NewRequirement("verrazzano.io/workload-type", selection.Equals, []string{"weblogic"})
	compReq, _ := labels.NewRequirement("app.oam.dev/component", selection.Equals, []string{wlName})
	appConfNameReq, _ := labels.NewRequirement("app.oam.dev/component", selection.Equals, []string{appConfig.Name})
	selector := labels.NewSelector()
	selector = selector.Add(*weblogicReq).Add(*compReq).Add(*appConfNameReq)

	var podList corev1.PodList
	if err := client.List(context.TODO(), &podList, &clipkg.ListOptions{Namespace: appConfig.Namespace, LabelSelector: selector}); err != nil {
		return err
	}

	// If any pod is using Isito 1.7.3 then stop the domain and return
	for _, pod := range podList.Items {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "proxyv2:1.7.3") {
				err := stopDomain(client, appConfig.Namespace, wlName)
				if err != nil {
					log.Errorf("Error annotating VerrazzanoWebLogicWorkload %s to stop the domain", wlName)
				}
				return err
			}
		}
	}
	return nil
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

// StartDomainsStoppedByUpgrade starts all the WebLogic domains that upgrade previously stopped
func StartDomainsStoppedByUpgrade(log *zap.SugaredLogger, client clipkg.Client) error {
	// get all the app configs
	appConfigs := oam.ApplicationConfigurationList{}
	if err := client.List(context.TODO(), &appConfigs, &clipkg.ListOptions{}); err != nil {
		log.Errorf("Error Listing appConfigs %v", err)
		return err
	}

	// Loop through the WebLogic workloads and start the ones that were stopped
	for _, appConfig := range appConfigs.Items {
		for _, wl := range appConfig.Status.Workloads {
			if wl.Reference.Kind == vzconst.VerrazzanoWebLogicWorkloadKind {
				if err := startDomainIfNeeded(client, appConfig.Namespace, wl.Reference.Name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Start the WebLogic domain if upgrade stopped it
func startDomainIfNeeded(client clipkg.Client, wlNamespace string, wlName string) error {
	// Set the lifecycle annotation on the VerrazzanoWebLogicWorkload
	var wl vzapp.VerrazzanoWebLogicWorkload
	wl.Namespace = wlNamespace
	wl.Name = wlName
	_, err := controllerutil.CreateOrUpdate(context.TODO(), client, &wl, func() error {
		if wl.ObjectMeta.Annotations == nil {
			return nil
		}
		if wl.ObjectMeta.Annotations[vzconst.LifecycleActionAnnotation] == vzconst.LifecycleActionStop {
			wl.ObjectMeta.Annotations[vzconst.LifecycleActionAnnotation] = vzconst.LifecycleActionStart
		}
		return nil
	})
	return err
}

// RestartAllApps restarts all the applications
func RestartAllApps(log *zap.SugaredLogger, client clipkg.Client, restartVersion string) error {
	// get all the app configs
	appConfigs := oam.ApplicationConfigurationList{}
	if err := client.List(context.TODO(), &appConfigs, &clipkg.ListOptions{}); err != nil {
		log.Errorf("Error Listing appConfigs %v", err)
		return err
	}

	for _, appConfig := range appConfigs.Items {
		// Set the update the restart version
		var ac oam.ApplicationConfiguration
		ac.Namespace = appConfig.Namespace
		ac.Name = appConfig.Name
		_, err := controllerutil.CreateOrUpdate(context.TODO(), client, &ac, func() error {
			if ac.ObjectMeta.Annotations == nil {
				ac.ObjectMeta.Annotations = make(map[string]string)
			}
			ac.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = restartVersion
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
