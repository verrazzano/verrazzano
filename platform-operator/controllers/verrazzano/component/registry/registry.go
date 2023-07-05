// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/argocd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	cmcontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/certmanager"
	cmconfig "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/issuer"
	cmocidns "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/webhookoci"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/clusteragent"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/clusterapi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/clusteroperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/console"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentbitosoutput"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafana"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafanadashboards"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	jaegeroperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/jaeger/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysqloperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearchdashboards"
	promadapter "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/adapter"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/kubestatemetrics"
	promnodeexporter "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/nodeexporter"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/pushgateway"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancherbackup"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/thanos"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/velero"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
)

type GetCompoentsFnType func() []spi.Component

var getComponentsFn = getComponents

var componentsRegistry []spi.Component

var getComponentsMap map[string]spi.Component

// OverrideGetComponentsFn Allows overriding the set of registry components for testing purposes
func OverrideGetComponentsFn(fnType GetCompoentsFnType) {
	getComponentsFn = fnType
	getComponentsMap = make(map[string]spi.Component)
}

// ResetGetComponentsFn Restores the GetComponents implementation to the default if it's been overridden for testing
func ResetGetComponentsFn() {
	getComponentsFn = getComponents
	getComponentsMap = make(map[string]spi.Component)
}

func InitRegistry() {
	componentsRegistry = []spi.Component{
		networkpolicies.NewComponent(), // This must be first, don't move it.  see netpol_components.go
		fluentoperator.NewComponent(),
		fluentbitosoutput.NewComponent(),
		oam.NewComponent(),
		appoper.NewComponent(),
		istio.NewComponent(),
		weblogic.NewComponent(),
		nginx.NewComponent(),
		cmcontroller.NewComponent(),
		cmocidns.NewComponent(),
		cmconfig.NewComponent(),
		externaldns.NewComponent(),
		clusterapi.NewComponent(),
		rancher.NewComponent(),
		verrazzano.NewComponent(),
		vmo.NewComponent(),
		opensearch.NewComponent(),
		opensearchdashboards.NewComponent(),
		grafana.NewComponent(),
		grafanadashboards.NewComponent(),
		authproxy.NewComponent(),
		coherence.NewComponent(),
		mysqloperator.NewComponent(), // mysqloperator needs to be upgraded before mysql
		mysql.NewComponent(),
		keycloak.NewComponent(),
		kiali.NewComponent(),
		promoperator.NewComponent(),
		promadapter.NewComponent(),
		kubestatemetrics.NewComponent(),
		pushgateway.NewComponent(),
		promnodeexporter.NewComponent(),
		jaegeroperator.NewComponent(),
		console.NewComponent(),
		fluentd.NewComponent(),
		velero.NewComponent(),
		rancherbackup.NewComponent(),
		clusteroperator.NewComponent(),
		argocd.NewComponent(),
		thanos.NewComponent(),
		clusteragent.NewComponent(),
	}
	getComponentsMap = make(map[string]spi.Component)
}

// GetComponents returns the list of components that are installable and upgradeable.
// The components will be processed in the order items in the array
func GetComponents() []spi.Component {
	if len(componentsRegistry) == 0 {
		InitRegistry()
	}
	return getComponentsFn()
}

// getComponents is the internal impl function for GetComponents, to allow overriding it for testing purposes
func getComponents() []spi.Component {
	return componentsRegistry
}

func FindComponent(componentName string) (bool, spi.Component) {
	// check if component is in map of looked up components
	existingComponent, ok := getComponentsMap[componentName]
	if !ok {
		for _, newComponent := range GetComponents() {
			if newComponent.Name() == componentName {
				getComponentsMap[componentName] = newComponent
				return true, newComponent
			}
		}
		// Component is not in registry
		return false, nil
	}
	return true, existingComponent
}

// ComponentDependenciesMet Checks if the declared dependencies for the component are ready and available; this is
// a shallow check of the direct dependencies, with the expectation that any indirect dependencies will be implicitly
// handled.
//
// For now, a dependency is soft; that is, we only care if it's Ready if it's enabled; if not we pass the dependency check
// so as not to block the dependent.  This would theoretically allow, for example, components that depend on Istio
// to continue to deploy if it's not enabled.  In the long run, the dependency mechanism should likely go away and
// allow components to individually make those decisions.
func ComponentDependenciesMet(c spi.Component, context spi.ComponentContext) bool {
	var notReadyDependencies []string
	var dependenciesReady = true
	log := context.Log()
	trace, err := checkDirectDependenciesReady(c, context, make(map[string]bool))
	if err != nil {
		log.Error(err.Error())
		return false
	}
	if len(trace) == 0 {
		log.Debugf("No dependencies declared for %s", c.Name())
		return true
	}
	log.Debugf("Trace results for %s: %v", c.Name(), trace)

	for compName, value := range trace {
		if !value {
			dependenciesReady = false
			notReadyDependencies = append(notReadyDependencies, compName)
		}
	}
	if !dependenciesReady {
		log.Progressf("Component %s waiting for dependencies %v to be ready", c.Name(), notReadyDependencies)
	}
	return dependenciesReady
}

// checkDependencies Check the ready state of any dependencies and check for cycles
func checkDirectDependenciesReady(c spi.Component, context spi.ComponentContext, stateMap map[string]bool) (map[string]bool, error) {
	compName := c.Name()
	log := context.Log()
	log.Debugf("Checking %s dependencies", compName)
	for _, dependencyName := range c.GetDependencies() {
		if compName == dependencyName {
			return stateMap, context.Log().ErrorfNewErr("Failed, illegal state, dependency cycle found for %s", c.Name())
		}
		if _, ok := stateMap[dependencyName]; ok {
			// dependency already checked
			log.Debugf("Dependency %s already checked", dependencyName)
			continue
		}
		found, dependency := FindComponent(dependencyName)
		if !found {
			return stateMap, context.Log().ErrorfNewErr("Failed, illegal state, declared dependency not found for %s: %s", c.Name(), dependencyName)
		}
		// Only check if dependency is ready when the dependency is enabled
		stateMap[dependencyName] = isDependencyReady(dependency, context) // dependency is ready
	}
	return stateMap, nil
}

// isDependencyReady Returns true if the component is disabled, is already in the Ready state, or if it's isReady() check is true
func isDependencyReady(dependency spi.Component, context spi.ComponentContext) bool {
	if !dependency.IsEnabled(context.EffectiveCR()) {
		return true
	}
	if isInReadyState(context, dependency) {
		// CR component status indicates ready
		return true
	}
	return dependency.IsReady(context)
}

func isInReadyState(context spi.ComponentContext, comp spi.Component) bool {
	if dependencyStatus, ok := context.ActualCR().Status.Components[comp.Name()]; ok {
		// We've already reported Ready status for this component
		if dependencyStatus.State == vzapi.CompStateReady {
			return true
		}
	}
	return false
}
