// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsworkload

import (
	"regexp"
	"strings"
	"testing"

	asserts "github.com/stretchr/testify/assert"
)

const log1 = `####<Nov 17, 2021 10:36:24,316 PM GMT> <Info> <Security> <bobs-bookstore-managed-server1> <> <main> <> <> <> <1637188584316> <[severity-value: 64] [partition-id: 0] [partition-name: DOMAIN] > <BEA-090905> <Disabling the CryptoJ JCE Provider self-integrity check for better startup performance. To enable this check, specify -Dweblogic.security.allowCryptoJDefaultJCEVerification=true.> 
`
const log2 = `####<Nov 17, 2021 5:55:46,955 PM GMT> <Trace> <com.oracle.logging.SystemLogger> <tododomain-adminserver> <AdminServer> <[ACTIVE] ExecuteThread: '9' for queue: 'weblogic.kernel.Default (self-tuning)'> <<WLS Kernel>> <> <cd1631dc-0279-48fc-bb29-85462f5ea85c-00000013> <1637171746955> <[severity-value: 256] [rid: 0] [partition-id: 0] [partition-name: DOMAIN] > <BEA-000000> <[com.oracle.cie.wls.config.online.Util:log] CIE Config Helper: registering MBean >>
####<Nov 17, 2021 5:55:46,955 PM GMT> <Trace> <com.oracle.logging.SystemLogger> <tododomain-adminserver> <AdminServer> <[ACTIVE] ExecuteThread: '9' for queue: 'weblogic.kernel.Default (self-tuning)'> <<WLS Kernel>> <> <cd1631dc-0279-48fc-bb29-85462f5ea85c-00000013> <1637171746955> <[severity-value: 256] [rid: 0] [partition-id: 0] [partition-name: DOMAIN] > <BEA-000000> <[com.oracle.cie.wls.config.online.Util:log] CIE Config Helper: isRegistered MBean: true> `

const log3 = `####<Nov 17, 2021 10:36:32,616 PM GMT> <Info> <RJVM> <bobs-bookstore-managed-server1> <> <Thread-11> <> <> <> <1637188592616> <[severity-value: 64] [partition-id: 0] [partition-name: DOMAIN] > <BEA-000570> <Network Configuration for Channel "managed-server1"
 Listen Address		 bobs-bookstore-managed-server1:8001
 Public Address		 N/A
 Http Enabled		 true
 Tunneling Enabled	 false
 Outbound Enabled	 false
 Admin Traffic Enabled	 true ResolveDNSName Enabled	 false>
`
const log4 = `####<Nov 17, 2021 10:37:35,553 PM GMT> <Info> <JDBC> <bobs-bookstore-managed-server1> <managed-server1> <[ACTIVE] ExecuteThread: '8' for queue: 'weblogic.kernel.Default (self-tuning)'> <<anonymous>> <> <c10cf642-4fbe-4753-9b2c-3c99b96df564-00000013> <1637188655553> <[severity-value: 64] [rid: 0] [partition-id: 0] [partition-name: DOMAIN] > <BEA-001156> <Stack trace associated with message 001129 follows: 

com.mysql.cj.jdbc.exceptions.CommunicationsException: Communications link failure

The last packet sent successfully to the server was 0 milliseconds ago. The driver has not received any packets from the server.
	at com.mysql.cj.jdbc.exceptions.SQLError.createCommunicationsException(SQLError.java:174)
	at com.mysql.cj.jdbc.exceptions.SQLExceptionsMapping.translateException(SQLExceptionsMapping.java:64)
	at com.mysql.cj.jdbc.ConnectionImpl.createNewIO(ConnectionImpl.java:835)
	at com.mysql.cj.jdbc.ConnectionImpl.<init>(ConnectionImpl.java:455)
	at com.mysql.cj.jdbc.ConnectionImpl.getInstance(ConnectionImpl.java:240)
	at com.mysql.cj.jdbc.NonRegisteringDriver.connect(NonRegisteringDriver.java:199)
	at weblogic.jdbc.common.internal.ConnectionEnvFactory.makeConnection0(ConnectionEnvFactory.java:321)
	at weblogic.jdbc.common.internal.ConnectionEnvFactory.access$000(ConnectionEnvFactory.java:20)
	at weblogic.jdbc.common.internal.ConnectionEnvFactory$1.run(ConnectionEnvFactory.java:219)
	at java.security.AccessController.doPrivileged(Native Method)
	at weblogic.jdbc.common.internal.ConnectionEnvFactory.makeConnection(ConnectionEnvFactory.java:216)
	at weblogic.jdbc.common.internal.ConnectionEnvFactory.setConnection(ConnectionEnvFactory.java:143)
	at weblogic.jdbc.common.internal.JDBCResourceFactoryImpl.createResource(JDBCResourceFactoryImpl.java:205)
	at weblogic.common.resourcepool.ResourcePoolImpl.makeResource(ResourcePoolImpl.java:1561)
	... 
	at weblogic.work.ExecuteThread.execute(ExecuteThread.java:420)
	at weblogic.work.ExecuteThread.run(ExecuteThread.java:360)
Caused By: com.mysql.cj.exceptions.CJCommunicationsException: Communications link failure

The last packet sent successfully to the server was 0 milliseconds ago. The driver has not received any packets from the server.
	at sun.reflect.NativeConstructorAccessorImpl.newInstance0(Native Method)
	...
	at com.mysql.cj.jdbc.ConnectionImpl.createNewIO(ConnectionImpl.java:825)
	at com.mysql.cj.jdbc.ConnectionImpl.<init>(ConnectionImpl.java:455)
	at com.mysql.cj.jdbc.ConnectionImpl.getInstance(ConnectionImpl.java:240)
	at com.mysql.cj.jdbc.NonRegisteringDriver.connect(NonRegisteringDriver.java:199)
	...
	at weblogic.work.ExecuteThread.run(ExecuteThread.java:360)
Caused By: java.net.UnknownHostException: mysql.wrong.svc.cluster.local: Temporary failure in name resolution
	at java.net.Inet6AddressImpl.lookupAllHostAddr(Native Method)
	...
	at com.mysql.cj.jdbc.ConnectionImpl.createNewIO(ConnectionImpl.java:825)
	at com.mysql.cj.jdbc.ConnectionImpl.<init>(ConnectionImpl.java:455)
	at com.mysql.cj.jdbc.ConnectionImpl.getInstance(ConnectionImpl.java:240)
	at com.mysql.cj.jdbc.NonRegisteringDriver.connect(NonRegisteringDriver.java:199)
    ...
	at weblogic.work.ExecuteThread.execute(ExecuteThread.java:420)
	at weblogic.work.ExecuteThread.run(ExecuteThread.java:360)
> 
`

// readFormat reads format1 ~ format13 in WlsFluentdParsingRules and combines format1 ~ format13 into one line
// - change ?< to ?P< to capture groups
// - use (?s) for new-line
func readFormat() string {
	format := false
	pattern := ""
	for _, line := range strings.Split(WlsFluentdParsingRules, "\n") {
		if strings.Contains(line, "format1 /") || format {
			format = true
			s := line[strings.Index(line, "/")+1 : strings.LastIndex(line, "/")]
			pattern = pattern + strings.ReplaceAll(s, "?<", "?P<")
			if strings.Contains(line, "format13 /") {
				return pattern + "(?s)"
			}
		}
	}
	return pattern
}

func Test_parseFormat(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string //expected text in the last line of the message
	}{
		{"log1", log1, "-Dweblogic.security.allowCryptoJDefaultJCEVerification=true."},
		{"log2", log2, "registering MBean >"},
		{"log3", log3, "Admin Traffic Enabled\t true ResolveDNSName Enabled\t false"},
		{"log4", log4, "java.net.UnknownHostException: mysql.wrong.svc.cluster.local"},
	}
	assert := asserts.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := regexp.MustCompile(readFormat())
			matches := reg.FindStringSubmatch(tt.text)
			//Total number of matches should be 27 = 1 + 13*2
			assert.Equal(27, len(matches))
			//The last match should be the message
			assert.Contains(matches[26], tt.expected)
		})
	}
}
