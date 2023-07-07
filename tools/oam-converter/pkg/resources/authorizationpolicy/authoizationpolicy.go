package authorizationpolicy

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	"istio.io/api/security/v1beta1"
	v1beta12 "istio.io/api/type/v1beta1"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

// creates Authorization Policy
func CreateAuthorizationPolicies(trait *vzapi.IngressTrait, rule vzapi.IngressRule, namePrefix string, hosts []string) error {
	// If any path needs an AuthorizationPolicy then add one for every path
	var addAuthPolicy bool
	for _, path := range rule.Paths {
		if path.Policy != nil {
			addAuthPolicy = true
		}
	}
	for _, path := range rule.Paths {
		if addAuthPolicy {
			requireFrom := true

			// Add a policy rule if one is missing
			if path.Policy == nil {
				path.Policy = &vzapi.AuthorizationPolicy{
					Rules: []*vzapi.AuthorizationRule{{}},
				}
				// No from field required, this is just a path being added
				requireFrom = false
			}

			pathSuffix := strings.Replace(path.Path, "/", "", -1)
			policyName := namePrefix
			if pathSuffix != "" {
				policyName = fmt.Sprintf("%s-%s", policyName, pathSuffix)
			}

			authzPolicy := &clisecurity.AuthorizationPolicy{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AuthorizationPolicy",
					APIVersion: consts.AuthorizationAPIVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      policyName,
					Namespace: constants.IstioSystemNamespace,
					Labels:    map[string]string{constants.LabelIngressTraitNsn: getIngressTraitNsn(trait.Namespace, trait.Name)},
				},
			}
			return mutateAuthorizationPolicy(authzPolicy, path.Policy, path.Path, hosts, requireFrom)
		}
	}
	return nil
}

// mutateAuthorizationPolicy changes the destination rule based upon a trait's configuration
func mutateAuthorizationPolicy(authzPolicy *clisecurity.AuthorizationPolicy, vzPolicy *vzapi.AuthorizationPolicy, path string, hosts []string, requireFrom bool) error {
	policyRules := make([]*v1beta1.Rule, len(vzPolicy.Rules))
	var err error
	for i, authzRule := range vzPolicy.Rules {
		policyRules[i], err = createAuthorizationPolicyRule(authzRule, path, hosts, requireFrom)
		if err != nil {
			print(err)
			return err
		}
	}

	authzPolicy.Spec = v1beta1.AuthorizationPolicy{
		Selector: &v1beta12.WorkloadSelector{
			MatchLabels: map[string]string{"istio": "ingressgateway"},
		},
		Rules: policyRules,
	}
	fmt.Println("AuthorizationPolicy", authzPolicy)
	directoryPath := "/Users/vrushah/GolandProjects/verrazzano/tools/oam-converter/"
	fileName := "az.yaml"
	filePath := filepath.Join(directoryPath, fileName)

	authzPolicyYaml, err := yaml.Marshal(authzPolicy)
	// Write the YAML content to the file
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		return err
	}
	defer file.Close()

	// Append the YAML content to the file
	_, err = file.Write(authzPolicyYaml)
	if err != nil {
		fmt.Printf("Failed to write to file: %v\n", err)
		return err
	}
	_, err = file.WriteString("---\n")
	return nil
}

// createAuthorizationPolicyRule uses the provided information to create an istio authorization policy rule
func createAuthorizationPolicyRule(rule *vzapi.AuthorizationRule, path string, hosts []string, requireFrom bool) (*v1beta1.Rule, error) {
	authzRule := v1beta1.Rule{}

	if requireFrom && rule.From == nil {
		return nil, fmt.Errorf("Authorization Policy requires 'From' clause")
	}
	if rule.From != nil {
		authzRule.From = []*v1beta1.Rule_From{
			{Source: &v1beta1.Source{
				RequestPrincipals: rule.From.RequestPrincipals},
			},
		}
	}

	if len(path) > 0 {
		authzRule.To = []*v1beta1.Rule_To{{
			Operation: &v1beta1.Operation{
				Hosts: hosts,
				Paths: []string{path},
			},
		}}
	}

	if rule.When != nil {
		conditions := []*v1beta1.Condition{}
		for _, vzCondition := range rule.When {
			condition := &v1beta1.Condition{
				Key:    vzCondition.Key,
				Values: vzCondition.Values,
			}
			conditions = append(conditions, condition)
		}
		authzRule.When = conditions
	}

	return &authzRule, nil
}

// Get Ingress Trait Namespace and name appended with "-"
func getIngressTraitNsn(namespace string, name string) string {
	return fmt.Sprintf("%s-%s", namespace, name)
}
