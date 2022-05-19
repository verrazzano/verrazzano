// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/istio"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	os2 "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	istiosec "istio.io/api/security/v1beta1"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// IstioCertSecret is the secret name used for Istio MTLS certs
	IstioCertSecret = "cacerts"

	istioTempPrefix           = "istio-"
	istioTempSuffix           = "yaml"
	istioTmpFileCreatePattern = istioTempPrefix + "*." + istioTempSuffix
	istioTmpFileCleanPattern  = istioTempPrefix + ".*\\." + istioTempSuffix
)

// create func vars for unit tests
type installFuncSig func(log vzlog.VerrazzanoLogger, imageOverridesString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error)

var installFunc installFuncSig = istio.Install

type forkInstallFuncSig func(compContext spi.ComponentContext, monitor installMonitor, overrideStrings string, files []string) error

var forkInstallFunc forkInstallFuncSig = forkInstall

type bashFuncSig func(inArgs ...string) (string, string, error)

var bashFunc bashFuncSig = os2.RunBash

func setInstallFunc(f installFuncSig) {
	installFunc = f
}

func setBashFunc(f bashFuncSig) {
	bashFunc = f
}

type installMonitorType struct {
	running         bool
	resultCh        chan bool
	inputCh         chan installRoutineParams
	istioctlSuccess bool
}

//installRoutineParams - Used to pass args to the install goroutine
type installRoutineParams struct {
	overrides     string
	fileOverrides []string
	log           vzlog.VerrazzanoLogger
}

//installMonitor - Represents a monitor object used by the component to monitor a background goroutine used for running
// istioctl install operations asynchronously.
type installMonitor interface {
	// checkResult - Checks for a result from the install goroutine; returns either the result of the operation, or an error indicating
	// the install is still in progress
	checkResult() (bool, error)
	// reset - Resets the monitor and closes any open channels
	reset()
	// isRunning - returns true of the monitor/goroutine are active
	isRunning() bool
	// run - Run the install with the specified args
	run(args installRoutineParams)
	// isIstioctlSuccess - returns boolean to indicate whether istioctl completed successfully
	isIstioctlSuccess() bool
}

//checkResult - checks for a result from the goroutine
// - returns false and a retry error if it's still running, or the result from the channel and nil if an answer was received
func (m *installMonitorType) checkResult() (bool, error) {
	select {
	case result := <-m.resultCh:
		m.istioctlSuccess = result
		return result, nil
	default:
		m.istioctlSuccess = false
		return false, ctrlerrors.RetryableError{Source: ComponentName}
	}
}

//reset - reset the monitor and close the channel
func (m *installMonitorType) reset() {
	m.running = false
	close(m.resultCh)
	close(m.inputCh)
}

//isRunning - returns true of the monitor/goroutine are active
func (m *installMonitorType) isRunning() bool {
	return m.running
}

//run - Run the install in a goroutine
func (m *installMonitorType) run(args installRoutineParams) {
	m.running = true
	m.resultCh = make(chan bool, 2)
	m.inputCh = make(chan installRoutineParams, 2)

	// Run the install in the background
	go func(inputCh chan installRoutineParams, outputCh chan bool) {
		// The function will execute once, sending true on success, false on failure to the channel reader
		// Read inputs
		args := <-inputCh
		log := args.log

		result := true
		m.istioctlSuccess = false
		log.Oncef("Component Istio is running istioctl")
		stdout, stderr, err := installFunc(log, args.overrides, args.fileOverrides...)
		log.Debugf("istioctl stdout: %s", string(stdout))
		if err != nil {
			result = false
			err = log.ErrorfNewErr("Failed calling istioctl install: %v stderr: %s", err.Error(), string(stderr))
		} else {
			log.Infof("Component Istio successfully ran istioctl install")
		}

		// Clean up the temp files
		removeTempFiles(log)

		// Write result
		outputCh <- result
	}(m.inputCh, m.resultCh)

	// Pass in the args to get started
	m.inputCh <- args
}

func (m *installMonitorType) isIstioctlSuccess() bool {
	return m.istioctlSuccess
}

func (i istioComponent) IsOperatorInstallSupported() bool {
	return true
}

// IsInstalled checks if Istio is installed by looking for the Istio control plane deployment
func (i istioComponent) IsInstalled(compContext spi.ComponentContext) (bool, error) {
	deployment := appsv1.Deployment{}
	nsn := types.NamespacedName{Name: IstiodDeployment, Namespace: IstioNamespace}
	if err := compContext.Client().Get(context.TODO(), nsn, &deployment); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		// Unexpected error
		return false, err
	}
	return true, nil
}

//Install - istioComponent install
//
// This utilizes the istioctl utility for install, which blocks during the entire installation process.  This can
// take up to several minutes and block the controller.  For now, we launch the install operation in a goroutine
// and requeue to check the result later.
//
// On subsequent callbacks, we check the status of the goroutine via the 'monitor' object.  If the monitor detects that
// the goroutine is still running, it returns a RetryableError that we return back to the controller to requeue and
// check again later.
//
// If the monitor detects that the goroutine is finished, we either return nil (success) for the successful install
// case, or reset the monitor state and drop down to the rest of the install method to retry the install again.
func (i istioComponent) Install(compContext spi.ComponentContext) error {
	if i.monitor.isRunning() {
		// Check the result
		succeeded, err := i.monitor.checkResult()
		if err != nil {
			// Not finished yet, requeue
			return err
		}
		// reset on success or failure
		i.monitor.reset()
		// If it's not finished running, requeue
		if succeeded {
			return nil
		}
		// if we were unsuccessful, reset and drop through to try again
		compContext.Log().Debug("Error during istio install, retrying")
	}

	var userFileCR *os.File
	var err error
	cr := compContext.EffectiveCR()
	log := compContext.Log()

	files := []string{i.ValuesFile}

	// Only create override file if the CR has an Istio component
	if cr.Spec.Components.Istio != nil {
		// create operator YAML
		istioOperatorYaml, err := BuildIstioOperatorYaml(compContext, cr.Spec.Components.Istio)
		if err != nil {
			return log.ErrorfNewErr("Failed to Build IstioOperator YAML: %v", err)
		}

		// Write the overrides to a tmp file
		userFileCR, err = ioutil.TempFile(os.TempDir(), istioTmpFileCreatePattern)
		if err != nil {
			return log.ErrorfNewErr("Failed to create temporary file for Istio install: %v", err)
		}
		if _, err = userFileCR.Write([]byte(istioOperatorYaml)); err != nil {
			return log.ErrorfNewErr("Failed to write to temporary file: %v", err)
		}
		if err := userFileCR.Close(); err != nil {
			return log.ErrorfNewErr("Failed to close temporary file: %v", err)
		}
		log.Debugf("Created values file from Istio install args: %s", userFileCR.Name())
		// append Operator YAML
		if userFileCR != nil {
			files = append(files, userFileCR.Name())
		}
	}

	overrideStrings, err := getOverridesString(compContext)
	if err != nil {
		return err
	}

	return forkInstallFunc(compContext, i.monitor, overrideStrings, files)
}

//forkInstall - istioctl install blocks, fork it into the background
func forkInstall(compContext spi.ComponentContext, monitor installMonitor, overrideStrings string, files []string) error {
	log := compContext.Log()
	log.Debugf("Creating background install goroutine for Istio")
	// clone the parameters
	overridesFilesCopy := make([]string, len(files))
	copy(overridesFilesCopy, files)

	// clone zap logger
	clone := log.GetZapLogger().With()
	log.SetZapLogger(clone)

	monitor.run(
		installRoutineParams{
			overrides:     overrideStrings,
			fileOverrides: overridesFilesCopy,
			log:           log,
		},
	)
	return ctrlerrors.RetryableError{Source: ComponentName}
}

func (i istioComponent) PreInstall(compContext spi.ComponentContext) error {
	if err := labelNamespace(compContext); err != nil {
		return err
	}
	if err := createCertSecret(compContext); err != nil {
		return err
	}
	return nil
}

func (i istioComponent) PostInstall(compContext spi.ComponentContext) error {
	if err := createPeerAuthentication(compContext); err != nil {
		return err
	}
	if err := createEnvoyFilter(compContext.Log(), compContext.Client()); err != nil {
		return err
	}
	return nil
}

func createCertSecret(compContext spi.ComponentContext) error {
	log := compContext.Log()
	if compContext.IsDryRun() {
		return nil
	}

	// Create the cert used by Istio MTLS if it doesn't exist
	var secret v1.Secret
	nsn := types.NamespacedName{Namespace: IstioNamespace, Name: IstioCertSecret}
	if err := compContext.Client().Get(context.TODO(), nsn, &secret); err != nil {
		if !errors.IsNotFound(err) {
			// Unexpected error
			return err
		}
		// Secret not found - create it
		certScript := filepath.Join(config.GetInstallDir(), "create-istio-cert.sh")
		if _, stderr, err := bashFunc(certScript); err != nil {
			return log.ErrorfNewErr("Failed creating Istio certificate secret %v: %s", err, stderr)
		}
	}
	return nil
}

// labelNamespace adds the label needed by network polices
func labelNamespace(compContext spi.ComponentContext) error {
	// Ensure Istio namespace exists and label it for network policies
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: IstioNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["verrazzano.io/namespace"] = IstioNamespace
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// createPeerAuthentication creates the PeerAuthentication resource to enable STRICT MTLS
func createPeerAuthentication(compContext spi.ComponentContext) error {
	peer := istioclisec.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: IstioNamespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &peer, func() error {
		if peer.Spec.Mtls == nil {
			peer.Spec.Mtls = &istiosec.PeerAuthentication_MutualTLS{}
		}
		peer.Spec.Mtls.Mode = istiosec.PeerAuthentication_MutualTLS_STRICT
		return nil
	})
	return err
}

func removeTempFiles(log vzlog.VerrazzanoLogger) {
	if err := os2.RemoveTempFiles(log.GetZapLogger(), istioTmpFileCleanPattern); err != nil {
		log.Errorf("Unexpected error removing temp files: %v", err.Error())
	}
}
