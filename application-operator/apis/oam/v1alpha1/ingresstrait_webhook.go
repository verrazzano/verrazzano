// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"fmt"
	s "strings"

	"k8s.io/apimachinery/pkg/runtime"
	k8sValidations "k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	c "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var getAllIngressTraits = listIngressTraits
var client c.Client

// log is for logging in this package.
var log = logf.Log.WithName("ingresstrait-resource")

// SetupWebhookWithManager saves client from manager and sets up webhook
func (r *IngressTrait) SetupWebhookWithManager(mgr ctrl.Manager) error {
	client = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:admissionReviewVersions={"v1","v1alpha1"},verbs=create;update,path=/validate-oam-verrazzano-io-v1alpha1-ingresstrait,mutating=false,failurePolicy=fail,groups=oam.verrazzano.io,resources=ingresstraits,sideEffects=None,versions=v1alpha1,name=vingresstrait.kb.io

var _ webhook.Validator = &IngressTrait{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for ingress trait type creation.
func (r *IngressTrait) ValidateCreate() error {
	log.Info("validate create", "name", r.Name)
	allIngressTraits, err := getAllIngressTraits(r.Namespace)
	if err != nil {
		return fmt.Errorf("unable to obtain list of existing IngressTrait's during create validation: %v", err)
	}
	return r.validateIngressTrait(allIngressTraits.Items)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for ingress trait type update.
func (r *IngressTrait) ValidateUpdate(old runtime.Object) error {
	log.Info("validate update", "name", r.Name)

	existingIngressList, err := getAllIngressTraits(r.Namespace)
	if err != nil {
		return fmt.Errorf("unable to obtain list of existing IngressTrait's during update validation: %v", err)
	}
	// Remove the trait that is being updated from the list
	updatedTrait := old.(*IngressTrait)
	updatedTraitUID := updatedTrait.UID
	allIngressTraits := existingIngressList.Items
	for i, existingTrait := range allIngressTraits {
		if existingTrait.UID == updatedTraitUID {
			allIngressTraits = append(allIngressTraits[:i], allIngressTraits[i+1:]...)
			break
		}
	}
	return r.validateIngressTrait(allIngressTraits)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for ingress trait type deletion.
func (r *IngressTrait) ValidateDelete() error {
	log.Info("validate delete", "name", r.Name)

	// no validation on delete
	return nil
}

// validateIngressTrait validates a new or updated ingress trait.
func (r *IngressTrait) validateIngressTrait(existingTraits []IngressTrait) error {
	// validation rules
	// For "exact" hosts such as "foo.example.com"
	// - ensure that no other ingressTrait exists with the same host and path
	// - For "prefix" hosts such as "*.example.com"
	// - These don't conflict with ingressTrait's with "exact" hosts as the exact host takes precedence because it is more specific
	// - Only conflict with other "prefix" ingressTraits with matching host string and path
	// For empty or * host
	// - * or empty host means to match all
	// - IngressTrait's with "all" hosts only conflict with other ingressTrait's with "all" hosts and same path
	// - "All" ingressTrait's don't conflict with "prefix" ingressTraits which take precedence because they are more specific
	// - "All" ingressTrait's don't conflict with "exact" ingressTraits which take precedence because they are more specific

	hostPathMap, e := r.createIngressTraitMap()
	if e != nil {
		return e
	}

	for _, ingressTrait := range existingTraits {
		for _, rule := range ingressTrait.Spec.Rules {
			hosts := getNormalizedHosts(rule)

			for _, host := range hosts {
				ingressPaths, exists := hostPathMap[host]
				if exists {
					for _, path := range rule.Paths {
						_, exists := ingressPaths[path.Path]
						if exists {
							return fmt.Errorf(
								"IngressTrait collision. An existing IngressTrait with the name: '%v' exists with host: '%v' and path: '%v'",
								ingressTrait.Name, host, path)
						}
					}
					// This is to support empty paths. We are considering defaulting to '/'.
					// With the '/' default, we can remove this block
					if len(rule.Paths) == 0 && len(ingressPaths) == 0 {
						return fmt.Errorf(
							"IngressTrait collision. An existing IngressTrait with the name: '%v' exists with host: '%v' and no paths",
							ingressTrait.Name, host)
					}
				}
			}
		}
	}
	return nil
}

// createIngressTraitMap creates a map of ingress traits with hosts mapped to associated paths.
func (r *IngressTrait) createIngressTraitMap() (map[string]map[string]struct{}, error) {
	hostPathMap := make(map[string]map[string]struct{})
	for _, rule := range r.Spec.Rules {
		hosts := getNormalizedHosts(rule)
		for _, host := range hosts {
			err := r.validateHost(host)
			if err != nil {
				return nil, err
			}
			paths, exists := hostPathMap[host]
			if !exists {
				paths = make(map[string]struct{})
				hostPathMap[host] = paths
			}
			for _, path := range rule.Paths {
				paths[path.Path] = struct{}{}
			}
		}
	}
	return hostPathMap, nil
}

// validateHost does syntactic validation of a host string
func (r *IngressTrait) validateHost(host string) error {
	if len(host) == 0 {
		return nil
	}

	if host == "*" {
		return nil
	}

	var errMessages []string
	var errFound bool

	if s.HasPrefix(host, "*.") {
		for _, msg := range k8sValidations.IsWildcardDNS1123Subdomain(host) {
			errMessages = append(errMessages, msg)
			errFound = true
		}
	} else {
		for _, msg := range k8sValidations.IsDNS1123Subdomain(host) {
			errMessages = append(errMessages, msg)
			errFound = true
		}
	}

	if !errFound {
		labels := s.Split(host, ".")
		for i := range labels {
			label := labels[i]
			// '*' isn't a valid label but is valid as the prefix in a wildcard host
			if !(i == 0 && label == "*") {
				for _, msg := range k8sValidations.IsDNS1123Label(label) {
					errMessages = append(errMessages, msg)
					errFound = true
				}
			}
		}
	}

	if errFound {
		return fmt.Errorf("invalid host specified for IngressTrait with name '%v': %v",
			r.Name, s.Join(errMessages, ", "))
	}
	return nil
}

// getNormalizedHosts gets a normalized host string from a rule
func getNormalizedHosts(rule IngressRule) []string {
	hosts := make([]string, len(rule.Hosts))
	for i, host := range rule.Hosts {
		host := s.TrimSpace(host)
		if host == "*" {
			host = ""
		}
		hosts[i] = host
	}
	return hosts
}

// listIngressTraits obtains all existing ingress traits in the specified namespace
func listIngressTraits(namespace string) (*IngressTraitList, error) {
	allIngressTraits := &IngressTraitList{}
	//todo: context.TODO or context.Background?
	//todo: currently ignoring namespace
	err := client.List(context.TODO(), allIngressTraits)
	return allIngressTraits, err
}
