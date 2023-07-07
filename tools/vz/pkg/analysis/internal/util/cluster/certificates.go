// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	encjson "encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"go.uber.org/zap"
)

// This is the initial entry function for certificate related issues.
// It first determines the status of the VPO, then checks if there are any certificates in the namespaces.
// It then analyzes those certificates to determine expiration or other issues and then contributes the respective issues to the Issue Reporter.
// The three issues that it is currently reporting on are the VPO hanging due to a long time to issues validate certificates, expired certificates, and when the certificate is not in a ready status.
func AnalyzeCertificateRelatedIssues(log *zap.SugaredLogger, clusterRoot string) (err error) {
	mapOfCertificatesInVPOToTheirNamespace, err := determineIfCLIIsHangingDueToCerts(log, clusterRoot)

	if err != nil {
		return err
	}
	allNamespacesFound, err = files.FindNamespaces(log, clusterRoot)
	if err != nil {
		return err
	}
	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}
	for _, namespace := range allNamespacesFound {
		certificateFile := files.FindFileInNamespace(clusterRoot, namespace, "certificates.json")
		certificateListForNamespace, err := getCertificateList(log, certificateFile)
		if err != nil {
			return err
		}
		if certificateListForNamespace == nil {
			continue
		}

		for _, certificate := range certificateListForNamespace.Items {
			if getLatestCondition(log, certificate) == nil {
				continue
			}
			conditionOfCert := getLatestCondition(log, certificate)
			if isCertConditionValid(conditionOfCert) && isVPOHangingonCert(mapOfCertificatesInVPOToTheirNamespace, certificate) {
				reportCLIHangingIssue(log, clusterRoot, certificate, &issueReporter, certificateFile)
				continue
			}
			if !(isCertConditionValid(conditionOfCert)) {
				reportGenericCertificateIssue(log, clusterRoot, certificate, &issueReporter, certificateFile)
				continue
			}
			if certificate.Status.NotAfter.Unix() < time.Now().Unix() {
				reportCertificateExpirationIssue(log, clusterRoot, certificate, &issueReporter, certificateFile)
			}

		}
	}
	issueReporter.Contribute(log, clusterRoot)
	return nil

}

// Checks if a condition of a certificate is valid
func isCertConditionValid(conditionOfCert *certv1.CertificateCondition) bool {
	return conditionOfCert.Status == "True" && conditionOfCert.Type == "Ready" && conditionOfCert.Message == "Certificate is up to date and has not expired"
}

// Determines if the VPO is currently hanging on a certificate
func isVPOHangingonCert(mapOfCertsThatVPOIsHangingOn map[string]string, certificate certv1.Certificate) bool {
	if len(mapOfCertsThatVPOIsHangingOn) <= 0 {
		return false
	}
	namespace, ok := mapOfCertsThatVPOIsHangingOn[certificate.ObjectMeta.Name]
	if ok && namespace == certificate.ObjectMeta.Namespace {
		return true
	}
	return false
}

// This function returns a list of certificate objects based on the certificates.json file
func getCertificateList(log *zap.SugaredLogger, path string) (certificateList *certv1.CertificateList, err error) {
	certList := &certv1.CertificateList{}
	file, err := os.Open(path)
	if err != nil {
		log.Debug("file %s not found", path)
		return nil, nil
	}
	defer file.Close()
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Error("Failed reading Json file %s", path)
		return nil, err
	}
	err = encjson.Unmarshal(fileBytes, &certList)
	if err != nil {
		log.Error("Failed to unmarshal CertificateList at %s", path)
		return nil, err
	}
	return certList, err
}

// This function returns the latest condition in a certificate, if one exists
func getLatestCondition(log *zap.SugaredLogger, certificate certv1.Certificate) *certv1.CertificateCondition {
	if certificate.Status.Conditions == nil {
		return nil
	}
	var latestCondition *certv1.CertificateCondition
	latestCondition = nil
	conditions := certificate.Status.Conditions
	for i, condition := range conditions {
		if condition.LastTransitionTime == nil {
			continue
		}
		if latestCondition == nil && condition.LastTransitionTime != nil {
			latestCondition = &(conditions[i])
			continue
		}
		if latestCondition.LastTransitionTime.UnixNano() < condition.LastTransitionTime.UnixNano() {
			latestCondition = &(conditions[i])
		}

	}
	return latestCondition
}

// This function reports when a VPO hanging issue has occurred

func reportCLIHangingIssue(log *zap.SugaredLogger, clusterRoot string, certificate certv1.Certificate, issueReporter *report.IssueReporter, certificateFile string) {
	files := []string{certificateFile}
	message := []string{fmt.Sprintf("The CLI is hanging due to a long time for the certificate to complete, but the certificate named %s in namespace %s is ready", certificate.ObjectMeta.Name, certificate.ObjectMeta.Namespace)}
	issueReporter.AddKnownIssueMessagesFiles(report.CLIHangingIssueDueToLongCertificateApproval, clusterRoot, message, files)

}

// This function reports if a certificate has expired
func reportCertificateExpirationIssue(log *zap.SugaredLogger, clusterRoot string, certificate certv1.Certificate, issueReporter *report.IssueReporter, certificateFile string) {
	files := []string{certificateFile}
	message := []string{fmt.Sprintf("The certificate named %s in namespace %s is expired", certificate.ObjectMeta.Name, certificate.ObjectMeta.Namespace)}
	issueReporter.AddKnownIssueMessagesFiles(report.CertificateExpired, clusterRoot, message, files)
}

// This function reports when a certificate is not expired, and the VPO is not hanging, but an issue has occurred.

func reportGenericCertificateIssue(log *zap.SugaredLogger, clusterRoot string, certificate certv1.Certificate, issueReporter *report.IssueReporter, certificateFile string) {
	files := []string{certificateFile}
	message := []string{fmt.Sprintf("The certificate named %s in namespace %s is not valid and experiencing issues", certificate.ObjectMeta.Name, certificate.ObjectMeta.Namespace)}
	issueReporter.AddKnownIssueMessagesFiles(report.CertificateExperiencingIssuesInCluster, clusterRoot, message, files)
}

// This function determines if the VPO is currently hanging due to certificate issues
// It does this by checking the last 10 logs of the VPO and determines all the certificates that the VPO is hanging along
// It returns a map containing these certificates as keys and their respective namespaces as values
// This map is used by the main certificate analysis function to determine if the VPO is hanging on a valid certificate
func determineIfCLIIsHangingDueToCerts(log *zap.SugaredLogger, clusterRoot string) (map[string]string, error) {
	listOfCertificatesThatCLIIsHangingOn := make(map[string]string)
	vpologRegExp := regexp.MustCompile(`verrazzano-install/verrazzano-platform-operator-.*/logs.txt`)
	allPodFiles, err := files.GetMatchingFiles(log, clusterRoot, vpologRegExp)
	if err != nil {
		return listOfCertificatesThatCLIIsHangingOn, err
	}
	if len(allPodFiles) == 0 {
		return listOfCertificatesThatCLIIsHangingOn, nil
	}
	vpoLog := allPodFiles[0]
	allMessages, err := files.ConvertToLogMessage(vpoLog)
	if err != nil {
		log.Error("Failed to convert files to the vpo message")
		return listOfCertificatesThatCLIIsHangingOn, err
	}
	//If the VPO has greater than 10 messages, the last 10 logs are the input. Else, the whole VPO logs are the input
	lastTenVPOLogs := allMessages[int(math.Max(float64(0), float64(len(allMessages)-10))):]
	//If the VPO has greater than 10 messages, the last 10 logs are the input. Else, the whole VPO logs are the input
	for _, VPOLog := range lastTenVPOLogs {
		VPOLogMessage := VPOLog.Message
		if strings.Contains(VPOLogMessage, "message: Issuing certificate as Secret does not exist") && strings.HasPrefix(VPOLogMessage, "Certificate ") {
			VPOLogCertificateNameAndNamespace := strings.Split(VPOLogMessage, " ")[1]
			namespaceAndCertificateNameSplit := strings.Split(VPOLogCertificateNameAndNamespace, "/")
			nameSpace := namespaceAndCertificateNameSplit[0]
			certificateName := namespaceAndCertificateNameSplit[1]
			_, ok := listOfCertificatesThatCLIIsHangingOn[certificateName]
			if !ok {
				listOfCertificatesThatCLIIsHangingOn[certificateName] = nameSpace
			}
		}

	}
	return listOfCertificatesThatCLIIsHangingOn, nil
}
