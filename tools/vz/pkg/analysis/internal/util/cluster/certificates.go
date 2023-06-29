// Copyright (c) 2023 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	encjson "encoding/json"
	"io"
	"os"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"go.uber.org/zap"
)

func AnalyzeCertificateRelatedIsssues(log *zap.SugaredLogger, clusterRoot string) (err error) {
	//First Step check if VPO Pod is detecting that a certificate expired
	allNamespacesFound, err = files.FindNamespaces(log, clusterRoot)
	if err != nil {
		return err
	}
	for _, namespace := range allNamespacesFound {
		certificateFile := files.FindFileInNamespace(clusterRoot, namespace, "certificates.json")
		certificateListForNamespace, err := getCertificateList(log, certificateFile)
		if err != nil {
			return err
		}
		var issueReporter = report.IssueReporter{
			PendingIssues: make(map[string]report.Issue),
		}
		for _, certificate := range certificateListForNamespace.Items {
			if certificate.Status.Conditions[len(certificate.Status.Conditions)-1].Status == "True" && certificate.Status.Conditions[len(certificate.Status.Conditions)-1].Type == "Ready" {

				//
			} else {
				reportCertificateIssue(log, clusterRoot, certificate, &issueReporter, certificateFile)
			}
		}
		issueReporter.Contribute(log, clusterRoot)

	}
	return nil
}

func getCertificateList(log *zap.SugaredLogger, path string) (certificateList *certv1.CertificateList, err error) {
	//I return nill here, as the namespace may not have a certificates.json file in it
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
func reportCertificateIssue(log *zap.SugaredLogger, clusterRoot string, certificate certv1.Certificate, issueReporter *report.IssueReporter, certificateFile string) {
	message := []string{"The certificate named " + certificate.ObjectMeta.Name + " in namespace " + certificate.ObjectMeta.Namespace + " is not ready or invalid"}
	files := []string{certificateFile}
	issueReporter.AddKnownIssueMessagesFiles("Certificate Not Ready/Invalid In Cluster", clusterRoot, message, files)
}
