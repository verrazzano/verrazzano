// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"

	runtimev1alpha1 "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultLoggingScopeLabel = "default.logging.scope"
	loggingScopeKind         = "LoggingScope"
	loggingScopeAPIVersion   = "oam.verrazzano.io/v1alpha1"
)

// LoggingScopeDefaulter supplies default LoggingScope
type LoggingScopeDefaulter struct {
	Client client.Client
}

// Default adds default LoggingScope to ApplicationConfiguration
func (d *LoggingScopeDefaulter) Default(appConfig *oamv1.ApplicationConfiguration, dryRun bool) (err error) {
	defaultLoggingComponentScope := oamv1.ComponentScope{
		ScopeReference: runtimev1alpha1.TypedReference{
			APIVersion: loggingScopeAPIVersion,
			Kind:       loggingScopeKind,
			Name:       getDefaultLoggingScopeName(appConfig),
		},
	}
	log.Info("defaultLoggingScope",
		"loggingScope", defaultLoggingComponentScope.ScopeReference.Name, "dryRun", dryRun)

	defaultScopeRequired := false
	for i := range appConfig.Spec.Components {
		if includeDefaultLoggingScope(&appConfig.Spec.Components[i], defaultLoggingComponentScope) {
			defaultScopeRequired = true
		}
	}
	if defaultScopeRequired {
		err = ensureDefaultLoggingScope(d.Client, appConfig, dryRun)
	} else {
		err = cleanupDefaultLoggingScope(d.Client, appConfig, dryRun)
	}
	return
}

// Cleanup cleans up the default logging scope associated with the given app config
func (d *LoggingScopeDefaulter) Cleanup(appConfig *oamv1.ApplicationConfiguration, dryRun bool) (err error) {
	err = cleanupDefaultLoggingScope(d.Client, appConfig, dryRun)
	return
}

// createDefaultLoggingScope creates the default logging scope for the given namespace
func createDefaultLoggingScope(name types.NamespacedName) *vzapi.LoggingScope {
	scope := &vzapi.LoggingScope{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    map[string]string{defaultLoggingScopeLabel: "true"},
		},
		Spec: vzapi.LoggingScopeSpec{
			WorkloadReferences: []runtimev1alpha1.TypedReference{},
		},
	}

	return scope
}

// includeDefaultLoggingScope updates the scopes of the given component to include the default logging scope, if appropriate
func includeDefaultLoggingScope(component *v1alpha2.ApplicationConfigurationComponent, defaultLoggingComponentScope oamv1.ComponentScope) (includeDefault bool) {
	includeDefault = true
	var scopes []oamv1.ComponentScope
	for _, scope := range component.Scopes {
		if scope.ScopeReference.Kind == loggingScopeKind {
			if scope.ScopeReference != defaultLoggingComponentScope.ScopeReference {
				includeDefault = false
				scopes = append(scopes, scope)
			}
		} else {
			scopes = append(scopes, scope)
		}
	}
	if includeDefault {
		scopes = append(scopes, defaultLoggingComponentScope)
	}
	component.Scopes = scopes
	return
}

// getDefaultLoggingScopeName gets the default logging scope name for a given app config
func getDefaultLoggingScopeName(appConfig *v1alpha2.ApplicationConfiguration) string {
	return fmt.Sprintf("default-%s-logging-scope", appConfig.Name)
}

// ensureDefaultLoggingScope checks that a default logging scope for the given app config exists and creates it if it doesn't
func ensureDefaultLoggingScope(c client.Client, appConfig *oamv1.ApplicationConfiguration, dryRun bool) (err error) {
	if !dryRun {
		defaultLoggingScopeName := getDefaultLoggingScopeName(appConfig)
		namespacedName := types.NamespacedName{Name: defaultLoggingScopeName, Namespace: appConfig.Namespace}
		var scope *vzapi.LoggingScope
		scope, err = fetchLoggingScope(context.TODO(), c, namespacedName)
		if scope == nil && err == nil {
			err = c.Create(
				context.TODO(),
				createDefaultLoggingScope(namespacedName),
				&client.CreateOptions{})
		}
	}
	return
}

// cleanupDefaultLoggingScope cleans up the default logging scope for the given app config
func cleanupDefaultLoggingScope(c client.Client, appConfig *oamv1.ApplicationConfiguration, dryRun bool) (err error) {
	if !dryRun {
		defaultLoggingScopeName := getDefaultLoggingScopeName(appConfig)
		namespacedName := types.NamespacedName{Name: defaultLoggingScopeName, Namespace: appConfig.Namespace}
		var scope *vzapi.LoggingScope
		scope, err = fetchLoggingScope(context.TODO(), c, namespacedName)
		if scope != nil && err == nil {
			if scope.Labels != nil && scope.Labels[defaultLoggingScopeLabel] == "true" {
				err = c.Delete(
					context.TODO(),
					createDefaultLoggingScope(namespacedName),
					&client.DeleteOptions{})
			}
		}
	}
	return
}

// fetchLoggingScope attempts to get a logging scope given a namespaced name
func fetchLoggingScope(ctx context.Context, c client.Reader, name types.NamespacedName) (*vzapi.LoggingScope, error) {
	log.Info("Fetch scope", "scope", name)
	var scope vzapi.LoggingScope
	err := c.Get(ctx, name, &scope)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("scope does not exist", "scope", name)
			return nil, nil
		}
		log.Info("failed to fetch scope", "scope", name)
	}
	return &scope, err
}
