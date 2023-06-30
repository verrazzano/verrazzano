// Copyright (c) 2023 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	encjson "encoding/json"
	"io"
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
func AnalyzeCertificateRelatedIsssues(log *zap.SugaredLogger, clusterRoot string) (err error) {
	mapOfCertificatesInVPOToTheirNamespace, err := determineIfVPOIsHangingDueToCerts(log, clusterRoot)

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
			if certificate.Status.Conditions[len(certificate.Status.Conditions)-1].Status == "True" && certificate.Status.Conditions[len(certificate.Status.Conditions)-1].Type == "Ready" && certificate.Status.Conditions[len(certificate.Status.Conditions)-1].Message == "Certificate is up to date and has not expired" {
				if len(mapOfCertificatesInVPOToTheirNamespace) > 0 {
					namespace, ok := mapOfCertificatesInVPOToTheirNamespace[certificate.ObjectMeta.Name]
					if ok && namespace == certificate.ObjectMeta.Namespace {
						reportCertificateIssue(log, clusterRoot, certificate, &issueReporter, certificateFile, true, false)
					}
				}
			} else if certificate.Status.NotAfter.Unix() < time.Now().Unix() {
				reportCertificateIssue(log, clusterRoot, certificate, &issueReporter, certificateFile, false, true)
			} else {
				reportCertificateIssue(log, clusterRoot, certificate, &issueReporter, certificateFile, false, false)
			}
		}
	}
	issueReporter.Contribute(log, clusterRoot)
	return nil

}

// This function returns a list of certificate objects based on the certificates.json file
func getCertificateList(log *zap.SugaredLogger, path string) (certificateList *certv1.CertificateList, err error) {
	certList := &certv1.CertificateList{}
	file, err := os.Open(path)
	if err != nil {
		log.Debugf("file %s not found", path)
		return nil, nil
	}
	defer file.Close()
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Debugf("Failed reading Json file %s", path)
		return nil, err
	}
	err = encjson.Unmarshal(fileBytes, &certList)
	if err != nil {
		log.Debugf("Failed to unmarshal CertificateList at %s", path)
		return nil, err
	}
	return certList, err
}

// This function creates the message and formally adds the issue to the IssueReporter
func reportCertificateIssue(log *zap.SugaredLogger, clusterRoot string, certificate certv1.Certificate, issueReporter *report.IssueReporter, certificateFile string, VPOHangingIssue bool, isCertificateExpired bool) {
	files := []string{certificateFile}
	if VPOHangingIssue {
		message := []string{"The VPO is hanging due to a long time for the certificate to complete, but the certificate named " + certificate.ObjectMeta.Name + " in namespace " + certificate.ObjectMeta.Namespace + " is ready"}
		issueReporter.AddKnownIssueMessagesFiles(report.VPOHangingIssueDueToLongCertificateApproval, clusterRoot, message, files)
		return
	}
	if isCertificateExpired {
		message := []string{"The certificate named " + certificate.ObjectMeta.Name + " in namespace " + certificate.ObjectMeta.Namespace + " is expired"}
		issueReporter.AddKnownIssueMessagesFiles(report.CertificateExpired, clusterRoot, message, files)
		return
	}
	message := []string{"The certificate named " + certificate.ObjectMeta.Name + " in namespace " + certificate.ObjectMeta.Namespace + " is not valid and experiencing issues"}
	issueReporter.AddKnownIssueMessagesFiles(report.CertificateExperiencingIssuesInCluster, clusterRoot, message, files)
}

// This function determines if the VPO is currently hanging due to certificate issues
// It does this by checking the last 10 logs of the VPO and determines all the certificates that the VPO is hanging along
// It returns a map containing these certificates as keys and their respective namespaces as values
// This map is used by the main certificate analysis function to determine if the VPO is hanging on a valid certificate
func determineIfVPOIsHangingDueToCerts(log *zap.SugaredLogger, clusterRoot string) (map[string]string, error) {
	listOfCertificatesThatVPOIsHangingOn := make(map[string]string)
	vpologRegExp := regexp.MustCompile(`verrazzano-install/verrazzano-platform-operator-.*/logs.txt`)
	allPodFiles, err := files.GetMatchingFiles(log, clusterRoot, vpologRegExp)
	if err != nil {
		return listOfCertificatesThatVPOIsHangingOn, err
	}
	if len(allPodFiles) == 0 {
		return listOfCertificatesThatVPOIsHangingOn, nil
	}
	vpoLog := allPodFiles[0]
	allMessages, err := files.ConvertToLogMessage(vpoLog)
	if err != nil {
		return listOfCertificatesThatVPOIsHangingOn, err
	}
	//If the VPO has greater than 10 messages, the last 10 logs are the input. Else, the whole VPO logs are the input
	lastTenVPOLogs := allMessages[:]
	if len(allMessages) > 10 {
		lastTenVPOLogs = allMessages[len(allMessages)-10:]
	}
	for _, VPOLog := range lastTenVPOLogs {
		VPOLogMessage := VPOLog.Message
		if strings.Contains(VPOLogMessage, "message: Issuing certificate as Secret does not exist") && strings.HasPrefix(VPOLogMessage, "Certificate ") {
			VPOLogCertificateNameAndNamespace := strings.Split(VPOLogMessage, " ")[1]
			namespaceAndCertificateNameSplit := strings.Split(VPOLogCertificateNameAndNamespace, "/")
			nameSpace := namespaceAndCertificateNameSplit[0]
			certificateName := namespaceAndCertificateNameSplit[1]
			_, ok := listOfCertificatesThatVPOIsHangingOn[certificateName]
			if !ok {
				listOfCertificatesThatVPOIsHangingOn[certificateName] = nameSpace
			}
		}

	}
	return listOfCertificatesThatVPOIsHangingOn, nil
}
