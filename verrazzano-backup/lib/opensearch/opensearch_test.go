package opensearch_test

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/constants"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/klog"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/opensearch"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/types"
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
	server *httptest.Server
	o      opensearch.Opensearch
)

func mockEnsureOpenSearchIsReachable(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Reachable ...")
	w.Header().Add("Content-Type", constants.HTTPContentType)
	w.WriteHeader(http.StatusOK)
	var osinfo types.OpenSearchClusterInfo
	osinfo.ClusterName = "foo"
	json.NewEncoder(w).Encode(osinfo)
}

func mockEnsureOpenSearchIsHealthy(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Healthy ...")
	w.Header().Add("Content-Type", constants.HTTPContentType)
	w.WriteHeader(http.StatusOK)
	var oshealth types.OpenSearchHealthResponse
	oshealth.ClusterName = "bar"
	oshealth.Status = "green"
	json.NewEncoder(w).Encode(oshealth)
}

func mockOpenSearchOperationResponse(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Snapshot register ...")
	w.Header().Add("Content-Type", constants.HTTPContentType)
	w.WriteHeader(http.StatusOK)
	var registerResponse types.OpenSearchOperationResponse
	registerResponse.Acknowledged = true
	json.NewEncoder(w).Encode(registerResponse)
}

func mockReloadOpensearchSecureSettings(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Reload secure settings")
	w.Header().Add("Content-Type", constants.HTTPContentType)
	w.WriteHeader(http.StatusOK)
	var reloadsettings types.OpenSearchSecureSettingsReloadStatus
	reloadsettings.ClusterNodes.Failed = 0
	reloadsettings.ClusterNodes.Total = 3
	reloadsettings.ClusterNodes.Successful = 3
	json.NewEncoder(w).Encode(reloadsettings)
}

func mockTriggerSnapshotRepository(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Snapshot ...")
	w.Header().Add("Content-Type", constants.HTTPContentType)
	w.WriteHeader(http.StatusOK)
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
	var array_ds []types.DataStreams
	var ds types.DataStreams
	ds.Name = "foo"
	ds.Status = constants.DataStreamGreen
	array_ds = append(array_ds, ds)
	ds.Name = "bar"
	array_ds = append(array_ds, ds)
	dsInfo.DataStreams = array_ds
	json.NewEncoder(w).Encode(dsInfo)

}

func TestMain(m *testing.M) {
	fmt.Println("Starting mock server")
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/":
			mockEnsureOpenSearchIsReachable(w, r)
		case "/_cluster/health":
			mockEnsureOpenSearchIsHealthy(w, r)
		case fmt.Sprintf("/_snapshot/%s", constants.OpeSearchSnapShotRepoName), fmt.Sprintf("/_data_stream/*"), fmt.Sprintf("/*"):
			mockOpenSearchOperationResponse(w, r)
		case fmt.Sprintf("/_nodes/reload_secure_settings"):
			mockReloadOpensearchSecureSettings(w, r)
		case fmt.Sprintf("/_snapshot/%s/%s", constants.OpeSearchSnapShotRepoName, "mango"), fmt.Sprintf("/_snapshot/%s/%s/_restore", constants.OpeSearchSnapShotRepoName, "mango"):
			mockTriggerSnapshotRepository(w, r)
		case fmt.Sprintf("/_data_stream"):
			mockRestoreProgress(w, r)

		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))

	fmt.Println("mock opensearch handler")
	timeParse, _ := time.ParseDuration("10m")
	o = opensearch.New(server.URL, timeParse, http.DefaultClient)

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

	var c types.ConnectionData
	c.BackupName = "mango"
	c.Timeout = "1s"
	err := o.EnsureOpenSearchIsReachable(&c, log)
	assert.Nil(t, err)
}

// Test_EnsureOpenSearchIsHealthy tests the EnsureOpenSearchIsHealthy method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN checks if opensearch cluster is healthy
func Test_EnsureOpenSearchIsHealthy(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	var c types.ConnectionData
	c.BackupName = "mango"
	c.Timeout = "1s"
	err := o.EnsureOpenSearchIsHealthy(&c, log)
	assert.Nil(t, err)
}

// Test_EnsureOpenSearchIsHealthy tests the EnsureOpenSearchIsHealthy method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN checks if opensearch cluster is healthy
func Test_RegisterSnapshotRepository(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

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
}

// Test_ReloadOpensearchSecureSettings tests the ReloadOpensearchSecureSettings method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN updates opensearch keystore creds
func Test_ReloadOpensearchSecureSettings(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)
	err := o.ReloadOpensearchSecureSettings(log)
	assert.Nil(t, err)
}

// TestTriggerSnapshot tests the TriggerSnapshot method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN creates a snapshot in object store
func Test_TriggerSnapshot(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	var c types.ConnectionData
	c.BackupName = "mango"
	err := o.TriggerSnapshot(&c, log)
	assert.Nil(t, err)
}

// TestCheckSnapshotProgress tests the CheckSnapshotProgress method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN tracks snapshot progress towards completion
func TestCheckSnapshotProgress(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

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

	err := o.DeleteData(log)
	assert.Nil(t, err)
}

// Test_TriggerSnapshot tests the TriggerRestore method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN creates a restore from object store from given snapshot name
func Test_TriggerRestore(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	var c types.ConnectionData
	c.BackupName = "mango"
	err := o.TriggerRestore(&c, log)
	assert.Nil(t, err)
}

// Test_CheckRestoreProgress tests the CheckRestoreProgress method for the following use case.
// GIVEN OpenSearch object
// WHEN invoked with snapshot name
// THEN tracks snapshot restore towards completion
func Test_CheckRestoreProgress(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

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
	err := o.Backup(&c, log)
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
	err := o.Restore(&c, log)
	assert.Nil(t, err)
}
