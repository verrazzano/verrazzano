// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// +k8s:openapi-gen=true
type Opss struct {
	// Name of a Secret containing the OPSS key wallet file, which must be in a field named walletFile. Use this to
	// allow a JRF domain to reuse its entries in the RCU database. This allows you to specify a wallet file that
	// was obtained from the domain home after the domain was booted for the first time.
	WalletFileSecret string `json:"walletFileSecret,omitempty"`

	// Name of a Secret containing the OPSS key passphrase, which must be in a field named walletPassword. Used to
	// encrypt and decrypt the wallet that is used for accessing the domain's entries in its RCU database.
	WalletPasswordSecret string `json:"walletPasswordSecret,omitempty"`
}
