package cluster

import (
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"go.uber.org/zap"
	"regexp"
)

func AnalyzeNetworkingIssues(log *zap.SugaredLogger, clusterRoot string) (err error) {
	if err != nil {
		return err
	}
	return nil
}
func determineIf(log *zap.SugaredLogger, clusterRoot string) (map[string]string, error) {
	listOfCertificatesThatVZClientIsHangingOn := make(map[string]string)
	vpologRegExp := regexp.MustCompile(`verrazzano-install/verrazzano-platform-operator-.*/logs.txt`)
	allPodFiles, err := files.GetMatchingFiles(log, clusterRoot, vpologRegExp)
	if err != nil {
		return listOfCertificatesThatVZClientIsHangingOn, err
	}
	if len(allPodFiles) == 0 {
		return listOfCertificatesThatVZClientIsHangingOn, nil
	}
	vpoLog := allPodFiles[0]
	allMessages, err := files.ConvertToLogMessage(vpoLog)
	if err != nil {
		log.Error("Failed to convert files to the vpo message")
		return listOfCertificatesThatVZClientIsHangingOn, err
	}
