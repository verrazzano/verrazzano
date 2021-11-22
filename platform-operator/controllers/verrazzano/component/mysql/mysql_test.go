// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package mysql

/*func TestCreateDBFile(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	fmt.Println(os.TempDir() + "/" + mysqlDBFile)
	err := createDBFile(spi.NewFakeContext(nil, vz, false, "../../../../manifests/profiles"))
	assert.Nil(t, err, "error creating db file")
}

func TestAppendOverrides(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	var devProfile vzapi.ProfileType = "dev"
	vz.Spec.Profile = devProfile
	ctx := spi.NewFakeContext(nil, vz, false, "../../../../manifests/profiles")
	var kvs []bom.KeyValue
	kvs, err := AppendMySQLOverrides(ctx, "", "", "", kvs)
	fmt.Println(kvs)
	assert.Nil(t, err, "Should be nil", err.Error())
}*/
