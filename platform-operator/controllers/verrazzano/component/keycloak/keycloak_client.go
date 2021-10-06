// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
    "crypto/tls"
	"fmt"
	"net/http"
	"net/url"
    "strings"
)

type KeycloakClient struct {
	client http.Client
	token  string
}

func (kc *KeycloakClient) Init(keycloakUrl string, username, password string) error {
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    client := &http.Client{Transport: tr}

	formArgs := url.Values{}
    formArgs.Set("grant_type", "password")
    formArgs.Set("scope",      "openid")
    formArgs.Set("client_id",  "admin-cli")
    formArgs.Set("username",   username)
    formArgs.Set("password",   password)

    //formArgs := fmt.Sprintf("grant_type=password\n&scope=openid\n&client_id=admin-cli\n&username=%s\n&password=%s", username, password)
    //formArgs := fmt.Sprintf("grant_type=password\n&scope=openid\n&client_id=admin-cli\n&username=%s\n&password=%s", username, password)
    //fmt.Printf("query string: %s\n", formArgs)
    //fmt.Printf("query string escaped: %s\n", url.QueryEscape(formArgs))
    //reader := strings.NewReader(url.QueryEscape(formArgs))

    reader := strings.NewReader(formArgs.Encode())

    buf2 := make([]byte, 1024)
    cntX, err := reader.Read(buf2)
    if cntX > 0 {
        fmt.Printf("reader read is: %s\n", string(buf2[0:cntX]))
    } else {
        fmt.Printf("reader is empty")
    }

    request, err := http.NewRequest(http.MethodPost, keycloakUrl, reader)
    if err != nil {
        fmt.Printf("Err is: %v\n", err)
        return err
    }

	resp, err := client.Do(request)
    if err != nil {
        fmt.Printf("Err is: %v\n", err)
        return err
    }
    fmt.Printf("HTTP error is: %s\n", resp.Status)

    buffer := make([]byte, 8192)
    cnt, err := resp.Body.Read(buffer)
    fmt.Printf("Body cnt is: %d\n", cnt)
    fmt.Printf("Err2 is: %v\n", err)
    if cnt > 0 {
        fmt.Printf("Body is: %s\n", string(buffer[1:cnt]))
    }
	return nil
}
