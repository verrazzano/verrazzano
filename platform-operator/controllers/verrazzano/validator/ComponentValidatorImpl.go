package validator

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
)

// Either
// -- Enhance the Component SPI to let each component validate the edit
// -- and/or just have a validator impl examine things directly


type ComponentValidatorImpl struct {}

var _ v1alpha1.ComponentValidator = ComponentValidatorImpl{}

func (c ComponentValidatorImpl) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
	for _, comp := range registry.GetComponents() {
		if err := comp.ValidateUpdate(old, new); err != nil {
			return err
		}
	}
	return nil
}
