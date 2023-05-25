package clusteragent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

// TestIsEnabled tests the IsEnabled function for the Cluster Agent
// GIVEN a Verrazzano CR
//
//	WHEN I call IsEnabled
//	THEN the function returns true if the cluster agent is enabled in the Verrazzano CR
func TestIsEnabled(t *testing.T) {
	a := assert.New(t)
	trueVal := true
	falseVal := false
	vzEmpty := &v1alpha1.Verrazzano{}
	vzEnabled := vzEmpty.DeepCopy()
	vzEnabled.Spec.Components.ClusterAgent = &v1alpha1.ClusterAgentComponent{
		Enabled: &trueVal,
	}
	vzDisabled := vzEmpty.DeepCopy()
	vzDisabled.Spec.Components.ClusterAgent = &v1alpha1.ClusterAgentComponent{
		Enabled: &falseVal,
	}

	component := NewComponent()
	a.True(component.IsEnabled(vzEmpty), "Expected empty cluster agent to return true")
	a.True(component.IsEnabled(vzEnabled), "Expected enabled cluster agent to return true")
	a.False(component.IsEnabled(vzDisabled), "Expected disabled cluster agent to return false")
}
