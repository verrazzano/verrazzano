// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch_test

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/constants"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/klog"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/opensearch"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/types"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/utils/vzk8sfake"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func logHelper() (*zap.SugaredLogger, string) {
	file, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("verrazzano-%s-hook-*.log", strings.ToLower("TEST")))
	if err != nil {
		fmt.Printf("Unable to create temp file")
		os.Exit(1)
	}
	defer file.Close()
	log, _ := klog.Logger(file.Name())
	return log, file.Name()
}

var (
	httpServer *httptest.Server
	openSearch opensearch.Opensearch
)

func mockEnsureOpenSearchIsReachable(error bool, w http.ResponseWriter, r *http.Request) {
	fmt.Println("Reachable ...")
	w.Header().Add("Content-Type", constants.HTTPContentType)
	if error {
		w.WriteHeader(http.StatusGatewayTimeout)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	var osinfo types.OpenSearchClusterInfo
	osinfo.ClusterName = "foo"
	json.NewEncoder(w).Encode(osinfo)
}

func mockEnsureOpenSearchIsHealthy(error bool, w http.ResponseWriter, r *http.Request) {
	fmt.Println("Healthy ...")
	w.Header().Add("Content-Type", constants.HTTPContentType)
	var oshealth types.OpenSearchHealthResponse
	oshealth.ClusterName = "bar"
	if error {
		w.WriteHeader(http.StatusGatewayTimeout)
		oshealth.Status = "red"
	} else {
		w.WriteHeader(http.StatusOK)
		oshealth.Status = "green"
	}
	json.NewEncoder(w).Encode(oshealth)
}

func mockOpenSearchOperationResponse(error bool, w http.ResponseWriter, r *http.Request) {
	fmt.Println("Snapshot register ...")
	w.Header().Add("Content-Type", constants.HTTPContentType)
	var registerResponse types.OpenSearchOperationResponse
	if error {
		w.WriteHeader(http.StatusGatewayTimeout)
		registerResponse.Acknowledged = false
	} else {
		w.WriteHeader(http.StatusOK)
		registerResponse.Acknowledged = true
	}

	json.NewEncoder(w).Encode(registerResponse)
}

func mockReloadOpensearchSecureSettings(error bool, w http.ResponseWriter, r *http.Request) {
	fmt.Println("Reload secure settings")
	w.Header().Add("Content-Type", constants.HTTPContentType)
	var reloadsettings types.OpenSearchSecureSettingsReloadStatus
	w.WriteHeader(http.StatusOK)
	reloadsettings.ClusterNodes.Total = 3
	if error {
		reloadsettings.ClusterNodes.Failed = 1
		reloadsettings.ClusterNodes.Successful = 3
	} else {
		reloadsettings.ClusterNodes.Failed = 0
		reloadsettings.ClusterNodes.Successful = 3
	}
	json.NewEncoder(w).Encode(reloadsettings)
}

func mockTriggerSnapshotRepository(error bool, w http.ResponseWriter, r *http.Request) {
	fmt.Println("Snapshot ...")
	w.Header().Add("Content-Type", constants.HTTPContentType)

	if error {
		w.WriteHeader(http.StatusGatewayTimeout)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	switch r.Method {
	case http.MethodPost:
		var triggerSnapshot types.OpenSearchSnapshotResponse
		triggerSnapshot.Accepted = true
		json.NewEncoder(w).Encode(triggerSnapshot)

	case http.MethodGet:
		var snapshotInfo types.OpenSearchSnapshotStatus
		var snapshots []types.Snapshot
		var snapshot types.Snapshot
		snapshot.Snapshot = "foo"
		snapshot.State = constants.OpenSearchSnapShotSucess
		snapshot.Indices = []string{"alpha", "beta", "gamma"}
		snapshot.DataStreams = []string{"mono", "di", "tri"}
		snapshots = append(snapshots, snapshot)
		snapshotInfo.Snapshots = snapshots
		json.NewEncoder(w).Encode(snapshotInfo)
	}

}

func mockRestoreProgress(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Restore progress ...")
	w.Header().Add("Content-Type", constants.HTTPContentType)
	w.WriteHeader(http.StatusOK)
	var dsInfo types.OpenSearchDataStreams
	var arrayDs []types.DataStreams
	var ds types.DataStreams
	ds.Name = "foo"
	ds.Status = constants.DataStreamGreen
	arrayDs = append(arrayDs, ds)
	ds.Name = "bar"
	arrayDs = append(arrayDs, ds)
	dsInfo.DataStreams = arrayDs
	json.NewEncoder(w).Encode(dsInfo)

}

func TestMain(m *testing.M) {
	fmt.Println("Starting mock server")
	httpServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/":
			mockEnsureOpenSearchIsReachable(false, w, r)
		case "/_cluster/health":
			mockEnsureOpenSearchIsHealthy(false, w, r)
		case fmt.Sprintf("/_snapshot/%s", constants.OpeSearchSnapShotRepoName), "/_data_stream/*", "/*":
			mockOpenSearchOperationResponse(false, w, r)
		case "/_nodes/reload_secure_settings":
			mockReloadOpensearchSecureSettings(false, w, r)
		case fmt.Sprintf("/_snapshot/%s/%s", constants.OpeSearchSnapShotRepoName, "mango"), fmt.Sprintf("/_snapshot/%s/%s/_restore", constants.OpeSearchSnapShotRepoName, "mango"):
			mockTriggerSnapshotRepository(false, w, r)
		case "/_data_stream":
			mockRestoreProgress(w, r)

		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer httpServer.Close()

	fmt.Println("mock opensearch handler")
	timeParse, _ := time.ParseDuration("10m")
	openSearch = opensearch.New(httpServer.URL, timeParse, http.DefaultClient)

	fmt.Println("Start tests")
	m.Run()
}

// Test_EnsureOpenSearchIsReachable tests the EnsureOpenSearchIsReachable method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with OpenSearch URL
// THEN verifies whether OpenSearch is reachable or not
func Test_EnsureOpenSearchIsReachable(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/":
			mockEnsureOpenSearchIsReachable(false, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server1.Close()
	timeParse, _ := time.ParseDuration("10m")
	o := opensearch.New(server1.URL, timeParse, http.DefaultClient)

	var c types.ConnectionData
	c.BackupName = "mango"
	c.Timeout = "1s"
	err := o.EnsureOpenSearchIsReachable(&c, log)
	assert.Nil(t, err)

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/":
			mockEnsureOpenSearchIsReachable(true, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server2.Close()

	o = opensearch.New(server2.URL, timeParse, http.DefaultClient)
	err = o.EnsureOpenSearchIsReachable(&c, log)
	assert.Nil(t, err)

}

// Test_EnsureOpenSearchIsHealthy tests the EnsureOpenSearchIsHealthy method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN checks if opensearch cluster is healthy
func Test_EnsureOpenSearchIsHealthy(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/_cluster/health":
			mockEnsureOpenSearchIsHealthy(false, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server1.Close()
	timeParse, _ := time.ParseDuration("10m")
	o := opensearch.New(server1.URL, timeParse, http.DefaultClient)

	var c types.ConnectionData
	c.BackupName = "mango"
	c.Timeout = "1s"
	err := o.EnsureOpenSearchIsHealthy(&c, log)
	assert.Nil(t, err)

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/_cluster/health":
			mockEnsureOpenSearchIsHealthy(true, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server2.Close()

	o = opensearch.New(server2.URL, timeParse, http.DefaultClient)
	err = o.EnsureOpenSearchIsHealthy(&c, log)
	assert.NotNil(t, err)
}

// Test_EnsureOpenSearchIsHealthy tests the EnsureOpenSearchIsHealthy method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN checks if opensearch cluster is healthy
func Test_RegisterSnapshotRepository(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case fmt.Sprintf("/_snapshot/%s", constants.OpeSearchSnapShotRepoName):
			mockOpenSearchOperationResponse(false, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server1.Close()
	timeParse, _ := time.ParseDuration("10m")
	o := opensearch.New(server1.URL, timeParse, http.DefaultClient)

	var objsecret types.ObjectStoreSecret
	objsecret.SecretName = "alpha"
	objsecret.SecretKey = "cloud"
	objsecret.ObjectAccessKey = "alphalapha"
	objsecret.ObjectSecretKey = "betabetabeta"
	var sdat types.ConnectionData
	sdat.Secret = objsecret
	sdat.BackupName = "mango"
	sdat.RegionName = "region"
	sdat.Endpoint = constants.OpenSearchURL

	err := o.RegisterSnapshotRepository(&sdat, log)
	assert.Nil(t, err)

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case fmt.Sprintf("/_snapshot/%s", constants.OpeSearchSnapShotRepoName):
			mockOpenSearchOperationResponse(true, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server2.Close()

	o = opensearch.New(server2.URL, timeParse, http.DefaultClient)
	err = o.RegisterSnapshotRepository(&sdat, log)
	assert.NotNil(t, err)

}

// Test_ReloadOpensearchSecureSettings tests the ReloadOpensearchSecureSettings method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN updates opensearch keystore creds
func Test_ReloadOpensearchSecureSettings(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/_nodes/reload_secure_settings":
			mockReloadOpensearchSecureSettings(false, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server1.Close()
	timeParse, _ := time.ParseDuration("10m")
	o := opensearch.New(server1.URL, timeParse, http.DefaultClient)
	err := o.ReloadOpensearchSecureSettings(log)
	assert.Nil(t, err)

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/_nodes/reload_secure_settings":
			mockReloadOpensearchSecureSettings(true, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server2.Close()
	o = opensearch.New(server2.URL, timeParse, http.DefaultClient)
	err = o.ReloadOpensearchSecureSettings(log)
	assert.NotNil(t, err)
}

// TestTriggerSnapshot tests the TriggerSnapshot method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN creates a snapshot in object store
func Test_TriggerSnapshot(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case fmt.Sprintf("/_snapshot/%s/%s", constants.OpeSearchSnapShotRepoName, "mango"):
			mockTriggerSnapshotRepository(false, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server1.Close()
	timeParse, _ := time.ParseDuration("10m")
	o := opensearch.New(server1.URL, timeParse, http.DefaultClient)

	var c types.ConnectionData
	c.BackupName = "mango"
	err := o.TriggerSnapshot(&c, log)
	assert.Nil(t, err)

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case fmt.Sprintf("/_snapshot/%s/%s", constants.OpeSearchSnapShotRepoName, "mango"):
			mockTriggerSnapshotRepository(true, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server2.Close()
	o = opensearch.New(server2.URL, timeParse, http.DefaultClient)

	err = o.TriggerSnapshot(&c, log)
	assert.NotNil(t, err)

}

// TestCheckSnapshotProgress tests the CheckSnapshotProgress method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN tracks snapshot progress towards completion
func TestCheckSnapshotProgress(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case fmt.Sprintf("/_snapshot/%s/%s", constants.OpeSearchSnapShotRepoName, "mango"):
			mockTriggerSnapshotRepository(false, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server.Close()
	timeParse, _ := time.ParseDuration("10m")
	o := opensearch.New(server.URL, timeParse, http.DefaultClient)

	var c types.ConnectionData
	c.BackupName = "mango"
	c.Timeout = "1s"
	err := o.CheckSnapshotProgress(&c, log)
	assert.Nil(t, err)
}

// Test_DeleteDataStreams tests the DeleteData method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with logger
// THEN deletes data from Opensearch cluster
func Test_DeleteDataStreams(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/_data_stream/*", "/*":
			mockOpenSearchOperationResponse(false, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server1.Close()
	timeParse, _ := time.ParseDuration("10m")
	o := opensearch.New(server1.URL, timeParse, http.DefaultClient)

	err := o.DeleteData(log)
	assert.Nil(t, err)

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/_data_stream/*", "/*":
			mockOpenSearchOperationResponse(true, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server2.Close()
	o = opensearch.New(server2.URL, timeParse, http.DefaultClient)

	err = o.DeleteData(log)
	assert.NotNil(t, err)
}

// Test_TriggerSnapshot tests the TriggerRestore method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN creates a restore from object store from given snapshot name
func Test_TriggerRestore(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case fmt.Sprintf("/_snapshot/%s/%s/_restore", constants.OpeSearchSnapShotRepoName, "mango"):
			mockTriggerSnapshotRepository(false, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server1.Close()
	timeParse, _ := time.ParseDuration("10m")
	o := opensearch.New(server1.URL, timeParse, http.DefaultClient)

	var c types.ConnectionData
	c.BackupName = "mango"
	c.Timeout = "1s"
	err := o.TriggerRestore(&c, log)
	assert.Nil(t, err)

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case fmt.Sprintf("/_snapshot/%s/%s/_restore", constants.OpeSearchSnapShotRepoName, "mango"):
			mockTriggerSnapshotRepository(true, w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server2.Close()
	o = opensearch.New(server2.URL, timeParse, http.DefaultClient)

	err = o.TriggerRestore(&c, log)
	assert.NotNil(t, err)
}

// Test_CheckRestoreProgress tests the CheckRestoreProgress method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN tracks snapshot restore towards completion
func Test_CheckRestoreProgress(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/_data_stream":
			mockRestoreProgress(w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server.Close()
	timeParse, _ := time.ParseDuration("10m")
	o := opensearch.New(server.URL, timeParse, http.DefaultClient)

	var c types.ConnectionData
	c.BackupName = "mango"
	c.Timeout = "1s"
	err := o.CheckRestoreProgress(&c, log)
	assert.Nil(t, err)
}

// Test_Backup tests the Backup method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN takes the opensearch backup
func Test_Backup(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	var c types.ConnectionData
	c.BackupName = "mango"
	c.Timeout = "1s"
	err := openSearch.Backup(&c, log)
	assert.Nil(t, err)
}

// Test_Restore tests the Restore method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN restores the opensearch from a given backup
func Test_Restore(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	var c types.ConnectionData
	c.BackupName = "mango"
	c.Timeout = "1s"
	err := openSearch.Restore(&c, log)
	assert.Nil(t, err)
}

// TestCheckDeployment tests the CheckDeployment method for the following use case.
// GIVEN k8s client
// WHEN restore is complete
// THEN checks kibana deployment is present on system
func Test_UpdateKeystore(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	var c types.ConnectionData
	c.BackupName = "mango"
	c.Timeout = "1s"

	var objsecret types.ObjectStoreSecret
	objsecret.SecretName = "alpha"
	objsecret.SecretKey = "cloud"
	objsecret.ObjectAccessKey = "alphalapha"
	objsecret.ObjectSecretKey = "betabetabeta"
	var sdat types.ConnectionData
	sdat.Secret = objsecret
	sdat.BackupName = "mango"
	sdat.RegionName = "region"
	sdat.Endpoint = constants.OpenSearchURL

	cfg, vzkfake := vzk8sfake.NewClientsetConfig()
	ok, err := openSearch.UpdateKeystore(vzkfake, cfg, &sdat, log)
	assert.Nil(t, err)
	assert.Equal(t, ok, true)

}
