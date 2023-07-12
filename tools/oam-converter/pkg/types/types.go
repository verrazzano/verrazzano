package types

import vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"

type ConversionComponents struct {
	AppName             string
	ComponentName       string
	AppNamespace        string
	IngressTrait        *vzapi.IngressTrait
	Helidonworkload     *vzapi.VerrazzanoHelidonWorkload
	Coherenceworkload   *vzapi.VerrazzanoCoherenceWorkload
	WeblogicworkloadMap map[string]*vzapi.VerrazzanoWebLogicWorkload
}
