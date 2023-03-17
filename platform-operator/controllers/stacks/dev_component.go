package stacks

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	helmcomp "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"
)

var chartDirPath = ""

type devComponent struct {
	helmcomp.HelmComponent
}

var _ spi.Component = devComponent{}

func newDevComponent(log vzlog.VerrazzanoLogger, cm v1.ConfigMap) (devComponent, error) {
	componentName, ok := cm.Data[componentNameKey]
	if !ok {
		annotationName := cm.Annotations[constants.VerrazzanoStackAnnotationName]
		log.Debugf("Component name field not included in ConfigMap %s data, defaulting to name %s from annotation", cm.Name, annotationName)
		componentName = annotationName
	}

	componentNamespace, ok := cm.Data[componentNamespaceKey]
	if !ok {
		log.Debugf("Component namespace field not included in ConfigMap %s data, defaulting to Configmap namespace", cm.Name)
		componentNamespace = cm.GetNamespace()
	}

	chartURL := cm.Data[chartURLKey]
	if chartURL == "" {
		return devComponent{}, fmt.Errorf("ConfigMap %s does not contain the chartURL field, cannot reconcile component %s", cm.Name, componentName)
	}
	return devComponent{
		helmcomp.HelmComponent{
			ReleaseName:             componentName,
			ChartDir:                filepath.Join(chartDirPath, chartURL),
			ChartNamespace:          componentNamespace,
			IgnoreNamespaceOverride: true,

			GetInstallOverridesFunc: func(_ runtime.Object) interface{} {
				return []v1alpha1.Overrides{{
					Values: &apiextensionsv1.JSON{
						Raw: []byte(cm.Data[overridesKey]),
					},
				}}
			},

			ImagePullSecretKeyname: constants.GlobalImagePullSecName,
		},
	}, nil
}

//func (h devComponent) Upgrade(context spi.ComponentContext) error {
//	// TODO: examine HelmComponent.Upgrade() to see what kind of hooks are missing/required
//	return h.Install(context)
//}

// IsReady Indicates whether a component is available and ready
func (h devComponent) IsReady(context spi.ComponentContext) bool {
	if context.IsDryRun() {
		context.Log().Debugf("IsReady() dry run for %s", h.ReleaseName)
		return true
	}

	// TODO: see if we need any of this nonsense below
	//releaseAppVersion, err := helm.GetReleaseAppVersion(h.ReleaseName, h.ChartNamespace)
	//if err != nil {
	//	return false
	//}
	//if h.ChartVersion != releaseAppVersion {
	//	return false
	//}

	if deployed, _ := helm.IsReleaseDeployed(h.ReleaseName, h.ChartNamespace); deployed {
		return true
	}
	return false
}

func InitChartDirPath() {
	chartDirPath = config.GetHelmChartsDir()
}
