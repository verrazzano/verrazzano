# Oracle Linux Cloud Native Environment Licensing Information User Manual

This chapter includes the software licensing information for additional Oracle software and third-party software products included as part of Oracle Linux Cloud Native Environment.

## 1.0 Licensing Information for Included Oracle Products

Other Oracle products distributed as part of Oracle Linux Cloud Native Environment are identified in the following table, along with the applicable licensing information.

**Table 1 License Overview for Included Oracle Products**

|Product|Subproduct|Licensing Information
|---------|-----------|-----------|
|Oracle Linux Cloud Native Environment |Oracle Linux |The Oracle Linux programs contain many components developed by Oracle and various third parties. The license for each component is located in this licensing documentation and/or in the component's source code. In addition, a list of components may be delivered with the Oracle Linux programs and the Additional Oracle Linux programs (as defined in the Oracle Linux License Agreement) or accessed online at https://oss.oracle.com/linux/legal/oracle-list.html. The source code for the Oracle Linux Programs and the Additional Oracle Linux programs can be found and accessed online at https://oss.oracle.com/sources/

## 2.0 Licensing Information for Software Packages and Container Images

- 2.1 Platform Api Server (Olcne-Lib)
- 2.2 Grafana
- 2.3 Prometheus
- 2.4 Helm

This section contains the licensing information for the Oracle Linux Cloud Native Environment software packages and container images.

### 2.1 Platform Api Server (Olcne-Lib)

**Table 2 Software Package Licensing Information for Platform Api Server (Olcne-Lib)**

|Component|RPM Packages|Container Images|License Information|Dependencies
|---------|-----------|-----------|-----------|-----------|
|olcne-lib |`olcnectl-1.1.2-6.el7.x86_64.rpm` `olcne-agent-1.1.2-6.el7.x86_64.rpm` `olcne-api-server-1.1.2-6.el7.x86_64.rpm` `olcne-utils-1.1.2-6.el7.x86_64.rpm` `olcne-nginx-1.1.2-6.el7.x86_64.rpm` `olcne-prometheus-chart-1.1.2-6.el7.x86_64.rpm` `olcne-istio-chart-1.1.2-6.el7.x86_64.rpm`| No container images.|[UPL](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/sw-licenses.html#comp-olcne-lib-1.1.2)| For software package dependency licensing information see Table 3, “Software Package Dependency Licensing Information for olcne-lib 1.1.2”.
|nginx |`nginx-container-image-1.12.2-1.0.1.el7.x86_64.rpm`|`nginx:1.12.2 `|[BSD](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/sw-licenses.html#comp-nginx-1.12.2)|Container images contain the Oracle Linux operating system which is covered by the Oracle Linux license as described in Section 1.0, “Licensing Information for Included Oracle Products”. For software package dependency licensing information see Table 4, “Software Package Dependency Licensing Information for nginx 1.12.2”.

**Table 3 Software Package Dependency Licensing Information for olcne-lib 1.1.2**

|Package Name|Version|License|
|---------|-----------|-----------|
|`github.com/AlecAivazis/survey/v2`|v2.0.1|[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_AlecAivazis_survey_v2-v2.0.1)
|`github.com/BurntSushi/toml`|v0.3.1|[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_BurntSushi_toml-v0.3.1)
|`github.com/MakeNowJust/heredoc`|e9091a26100e|[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_MakeNowJust_heredoc-e9091a26100e)
|`github.com/Masterminds/semver`|v1.4.2 | [MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_Masterminds_semver-v1.4.2)
|`github.com/Microsoft/go-winio`|v0.4.14 |[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_Microsoft_go-winio-v0.4.14)
|`github.com/coreos/etcd`|v3.3.10 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_coreos_etcd-v3.3.10)
|`github.com/evanphx/json-patch`|v4.5.0 	| [BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_evanphx_json-patch-v4.5.0)
|`github.com/gogo/protobuf`|v1.3.1|https://github.com/gogo/protobuf/blob/master/LICENSE
|`github.com/golang/groupcache`|611e8accdfc9 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_golang_groupcache-611e8accdfc9)
|`github.com/golang/mock`|v1.3.1 	|[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_golang_mock-v1.3.1)
|`github.com/golang/protobuf`|v1.3.2 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_golang_protobuf-v1.3.2)
|`github.com/google/gofuzz`|v1.1.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_google_gofuzz-v1.1.0)
|`github.com/google/uuid`|v1.1.1 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_google_uuid-v1.1.1)
|`github.com/googleapis/gnostic`|v0.3.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_googleapis_gnostic-v0.3.0)
|`github.com/hashicorp/vault/api`|v1.0.4 |	[Mozilla Public License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_hashicorp_vault_api-v1.0.4)
|`github.com/hinshun/vt10x`|d55458df857c |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_hinshun_vt10x-d55458df857c)
|`github.com/imdario/mergo`|v0.3.7 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_imdario_mergo-v0.3.7)
|`github.com/json-iterator/go`|v1.1.9 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_json-iterator_go-v1.1.9)
|`github.com/kr/pty`|v1.1.8 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_kr_pty-v1.1.8)
|`github.com/pkg/errors`|v0.8.1 |	[BSD 2-Clause "Simplified" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_pkg_errors-v0.8.1)
|`github.com/rifflock/lfshook`|b9218ef580f5 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_rifflock_lfshook-b9218ef580f5)
|`github.com/sirupsen/logrus`|v1.4.2 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_sirupsen_logrus-v1.4.2)
|`github.com/spf13/cobra`|v0.0.5 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_spf13_cobra-v0.0.5)
|`github.com/spf13/pflag`|v1.0.5 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_spf13_pflag-v1.0.5)
|`github.com/stretchr/testify`|v1.4.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_stretchr_testify-v1.4.0)
|`golang.org/x/net`|13f9640d40b9 |	https://github.com/golang/net/blob/master/LICENSE
|`golang.org/x/tools`|6de373a2766c |	https://github.com/golang/tools/blob/master/LICENSE
|`google.golang.org/genproto`|3bdd9d9f5532 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#google.golang.org_genproto-3bdd9d9f5532)
|`google.golang.org/grpc`|v1.23.1 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#google.golang.org_grpc-v1.23.1)
|`gopkg.in/yaml.v2`|v2.2.8 |	https://github.com/go-yaml/yaml/blob/v2/LICENSE
|`k8s.io/api`|v0.17.4 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_api-v0.17.4)
|`k8s.io/apiextensions-apiserver`|v0.0.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_apiextensions-apiserver-v0.0.0)
|`k8s.io/apimachinery`|v0.17.4 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_apimachinery-v0.17.4)
|`k8s.io/apiserver`|v0.17.4 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_apiserver-v0.17.4)
|`k8s.io/client-go`|8e4128053008 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_client-go-8e4128053008)
|`k8s.io/cloud-provider`|v0.17.4 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_cloud-provider-v0.17.4)
|`k8s.io/cluster-bootstrap`|v0.0.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_cluster-bootstrap-v0.0.0)
|`k8s.io/component-base`|v0.17.4 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_component-base-v0.17.4)
|`k8s.io/kube-openapi`|addea2498afe |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_kube-openapi-addea2498afe)
|`k8s.io/kube-proxy`|v0.0.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_kube-proxy-v0.0.0)
|`k8s.io/kubelet`|v0.0.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_kubelet-v0.0.0)
|`k8s.io/kubernetes`|v1.17.4 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_kubernetes-v1.17.4)
|`k8s.io/utils`|861946025e34 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_utils-861946025e34)
|`sigs.k8s.io/yaml`|v1.2.0 |	https://github.com/ghodss/yaml/blob/master/LICENSE

**Table 4 Software Package Dependency Licensing Information for nginx 1.12.2**

|Package Name|Version|License|
|---------|-----------|-----------|
|`pcre`|8.32|[BSD](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#pcre-8.32)
|`openssl`|1.0.2k|[Apache-2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#openssl-1.0.2k)
|`zlib`|1.2.7|[BSD](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#zlib-1.2.7)

### 2.2 Grafana

**Table 5 Software Package Licensing Information for Grafana**

|Component|RPM Packages|Container Images|License Information|Dependencies
|---------|-----------|-----------|-----------|-----------|
|grafana |`grafana-6.7.4-1.0.1.el7.x86_64.rpm`|`grafana:v6.7.4` |[Apache-2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/sw-licenses.html#comp-grafana-6.7.4)|Container images contain the Oracle Linux operating system which is covered by the Oracle Linux license as described in Section 1.0, “Licensing Information for Included Oracle Products”. For software package dependency licensing information see Table 6, “Software Package Dependency Licensing Information for grafana 6.7.4”.

**Table 6 Software Package Dependency Licensing Information for grafana 6.7.4**

|Package Name|Version|License|
|---------|-----------|-----------|
|`cloud.google.com/go` |	v0.38.0 |	[Apache-2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#cloud.google.com_go-v0.38.0)
|`github.com/BurntSushi/toml` |	v0.3.1 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_BurntSushi_toml-v0.3.1)
|`github.com/VividCortex/mysqlerr` |	6c6b55f8796f |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_VividCortex_mysqlerr-6c6b55f8796f)
|`github.com/aws/aws-sdk-go` |	v1.25.48 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_aws_aws-sdk-go-v1.25.48)
|`github.com/beevik/etree` |	v1.1.0 |	[BSD-2-Clause](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_beevik_etree-v1.1.0)
|`github.com/benbjohnson/clock` |	7dc76406b6d3 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_benbjohnson_clock-7dc76406b6d3)
|`github.com/bradfitz/gomemcache` |	551aad21a668 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_bradfitz_gomemcache-551aad21a668)
|`github.com/codahale/hdrhistogram`|	3a0bb77429bd |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_codahale_hdrhistogram-3a0bb77429bd)
|`github.com/crewjam/saml` |	c42136edf9b1 |	[BSD 2-Clause "Simplified" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_crewjam_saml-c42136edf9b1)
|`github.com/davecgh/go-spew` |	v1.1.1 |	[ISC License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_davecgh_go-spew-v1.1.1)
|`github.com/denisenkom/go-mssqldb` |	a8ed825ac853 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_denisenkom_go-mssqldb-a8ed825ac853)
|`github.com/facebookgo/ensure` |	b4ab57deab51 |	[BSD-3-Clause](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_facebookgo_ensure-b4ab57deab51)
|`github.com/facebookgo/inject` |	f23751cae28b |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_facebookgo_inject-f23751cae28b)
|`github.com/facebookgo/stack` |	751773369052 |	[BSD-3-Clause](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_facebookgo_stack-751773369052)
|`github.com/facebookgo/structtag` |	217e25fb9691 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_facebookgo_structtag-217e25fb9691)
|`github.com/facebookgo/subset` |	8dac2c3c4870 |	[BSD-3-Clause](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_facebookgo_subset-8dac2c3c4870)
|`github.com/fatih/color` |	v1.7.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_fatih_color-v1.7.0)
|`github.com/go-macaron/binding` |	0b4f37bab25b |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_go-macaron_binding-0b4f37bab25b)
|`github.com/go-macaron/gzip` |	cad1c6580a07 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_go-macaron_gzip-cad1c6580a07)
|`github.com/go-macaron/session` |	1a3cdc6f5659 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_go-macaron_session-1a3cdc6f5659)
|`github.com/go-sql-driver/mysql` |	v1.4.1 |	[Mozilla Public License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_go-sql-driver_mysql-v1.4.1)
|`github.com/go-stack/stack` |	v1.8.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_go-stack_stack-v1.8.0)
|`github.com/go-xorm/core` |	v0.6.2 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_go-xorm_core-v0.6.2)
|`github.com/go-xorm/xorm` |	v0.7.1 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_go-xorm_xorm-v0.7.1)
|`github.com/gobwas/glob` |	v0.2.3 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_gobwas_glob-v0.2.3)
|`github.com/google/go-cmp` |	v0.3.1 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_google_go-cmp-v0.3.1)
|`github.com/gorilla/websocket` |	v1.4.1 |	[BSD 2-Clause "Simplified" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_gorilla_websocket-v1.4.1)
|`github.com/gosimple/slug` |	v1.4.2 |	[Mozilla Public License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_gosimple_slug-v1.4.2)
|`github.com/grafana/grafana-plugin-model` |	1fc953a61fb4 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_grafana_grafana-plugin-model-1fc953a61fb4)
|`github.com/grafana/grafana-plugin-sdk-go` |	v0.30.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_grafana_grafana-plugin-sdk-go-v0.30.0)
|`github.com/hashicorp/go-hclog` |	ff2cf002a8dd |[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_hashicorp_go-hclog-ff2cf002a8dd)
|`github.com/hashicorp/go-plugin`	|v1.0.1 |[Mozilla Public License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_hashicorp_go-plugin-v1.0.1)
|`github.com/hashicorp/go-version` |	v1.1.0 |	[Mozilla Public License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_hashicorp_go-version-v1.1.0)
|`github.com/inconshreveable/log15` |	67afb5ed74ec |	[Apache-2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_inconshreveable_log15-67afb5ed74ec)
|`github.com/jmespath/go-jmespath` |	c2b33e8439af |	[Apache-2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_jmespath_go-jmespath-c2b33e8439af)
|`github.com/jung-kurt/gofpdf` |	v1.10.1 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_jung-kurt_gofpdf-v1.10.1)
|`github.com/k0kubun/colorstring` |	9440f1994b88 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_k0kubun_colorstring-9440f1994b88)
|`github.com/klauspost/compress` |	v1.4.1 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_klauspost_compress-v1.4.1)
|`github.com/klauspost/cpuid` |	v1.2.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_klauspost_cpuid-v1.2.0)
|`github.com/lib/pq` |	v1.2.0 |	[MIT](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_lib_pq-v1.2.0)
|`github.com/linkedin/goavro/v2` |	v2.9.7 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_linkedin_goavro_v2-v2.9.7)
|`github.com/mattn/go-colorable` |	v0.1.6 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_mattn_go-colorable-v0.1.6)
|`github.com/mattn/go-isatty` |	v0.0.12 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_mattn_go-isatty-v0.0.12)
|`github.com/mattn/go-sqlite3` |	v1.11.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_mattn_go-sqlite3-v1.11.0)
|`github.com/opentracing/opentracing-go` |	v1.1.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_opentracing_opentracing-go-v1.1.0)
|`github.com/patrickmn/go-cache` |	v2.1.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_patrickmn_go-cache-v2.1.0)
|`github.com/pkg/errors` |	v0.8.1 |	[BSD 2-Clause "Simplified" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_pkg_errors-v0.8.1)
|`github.com/prometheus/client_golang` |	v1.3.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_prometheus_client_golang-v1.3.0)
|`github.com/prometheus/client_model` |	v0.1.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_prometheus_client_model-v0.1.0)
|`github.com/prometheus/common` |	v0.7.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_prometheus_common-v0.7.0)
|`github.com/rainycape/unidecode` |	cb7f23ec59be |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_rainycape_unidecode-cb7f23ec59be)
|`github.com/robfig/cron` |	b41be1df6967 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_robfig_cron-b41be1df6967)
|`github.com/robfig/cron/v3` |	v3.0.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_robfig_cron_v3-v3.0.0)
|`github.com/sergi/go-diff` |	v1.0.0 |	[MIT](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_sergi_go-diff-v1.0.0)
|`github.com/smartystreets/goconvey` |	505e41936337 |	[UNKNOWN_LICENSE_TYPE](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_smartystreets_goconvey-505e41936337)
|`github.com/stretchr/testify` |	v1.4.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_stretchr_testify-v1.4.0)
|`github.com/teris-io/shortid` |	771a37caa5cf |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_teris-io_shortid-771a37caa5cf)
|`github.com/ua-parser/uap-go` |	daf92ba38329 |	[Apache-2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_ua-parser_uap-go-daf92ba38329)
|`github.com/uber/jaeger-client-go` |	v2.20.1 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_uber_jaeger-client-go-v2.20.1)
|`github.com/uber/jaeger-lib` |	v2.2.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_uber_jaeger-lib-v2.2.0)
|`github.com/unknwon/com` |	v1.0.1 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_unknwon_com-v1.0.1)
|`github.com/urfave/cli/v2` |	v2.1.1 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_urfave_cli_v2-v2.1.1)
|`github.com/yudai/gojsondiff` |	v1.0.0| 	[MIT](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_yudai_gojsondiff-v1.0.0)
|`github.com/yudai/golcs` |	ecda9a501e82 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_yudai_golcs-ecda9a501e82)
|`github.com/yudai/pp` |	v2.0.1 |	Unknown
|`go.uber.org/atomic` |	v1.5.1 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#go.uber.org_atomic-v1.5.1)
|`golang.org/x/crypto` |	87dc89f01550 |	[BSD-3-Clause](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#golang.org_x_crypto-87dc89f01550)
|`golang.org/x/lint` |	fdd1cda4f05f 	|[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#golang.org_x_lint-fdd1cda4f05f)
|`golang.org/x/net` |	aa69164e4478 |	[BSD-3-Clause](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#golang.org_x_net-aa69164e4478)
|`golang.org/x/oauth2` |	0f29369cfe45 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#golang.org_x_oauth2-0f29369cfe45)
|`golang.org/x/sync` |	cd5d95a43a6e |	[BSD-3-Clause](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#golang.org_x_sync-cd5d95a43a6e)
|`golang.org/x/tools` |	04c2e8eff935 |	[BSD-3-Clause](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#golang.org_x_tools-04c2e8eff935)
|`golang.org/x/xerrors` |	1b5146add898 |	[BSD-3-Clause](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#golang.org_x_xerrors-1b5146add898)
|`google.golang.org/genproto` |	54afdca5d873 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#google.golang.org_genproto-54afdca5d873)
`gopkg.in/alexcesaro/quotedprintable.v3` |	2caba252f4dc |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#gopkg.in_alexcesaro_quotedprintable.v3-2caba252f4dc)
|`google.golang.org/grpc` |	v1.23.1 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#google.golang.org_grpc-v1.23.1)
|`gopkg.in/asn1-ber.v1` |	f715ec2f112d |	[MIT](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#gopkg.in_asn1-ber.v1-f715ec2f112d)
|`gopkg.in/ini.v1` |	v1.46.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#gopkg.in_ini.v1-v1.46.0)
|`gopkg.in/ldap.v3` |	v3.0.2 |	[MIT](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#gopkg.in_ldap.v3-v3.0.2)
|`gopkg.in/macaron.v1` |	v1.3.4 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#gopkg.in_macaron.v1-v1.3.4)
|`gopkg.in/mail.v2` |	v2.3.1 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#gopkg.in_mail.v2-v2.3.1)
|`gopkg.in/redis.v5` |	v5.2.9 |	[BSD 2-Clause "Simplified" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#gopkg.in_redis.v5-v5.2.9)
|`gopkg.in/square/go-jose.v2` |	v2.4.1 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#gopkg.in_square_go-jose.v2-v2.4.1)
|`gopkg.in/yaml.v2` |	v2.2.5 |	[Apache-2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#gopkg.in_yaml.v2-v2.2.5)

### 2.3 Prometheus

**Table 7 Software Package Licensing Information for Prometheus**

|Component|RPM Packages|Container Images|License Information|Dependencies
|---------|-----------|-----------|-----------|-----------|
|prometheus |`prometheus-2.13.1-1.0.2.el7.x86_64.rpm `|`prometheus:v2.13.1 `|[Apache-2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/sw-licenses.html#comp-prometheus-2.13.1)|Container images contain the Oracle Linux operating system which is covered by the Oracle Linux license as described in Section 1.0, “Licensing Information for Included Oracle Products”. For software package dependency licensing information see Table 8, “Software Package Dependency Licensing Information for prometheus 2.13.1”

**Table 8 Software Package Dependency Licensing Information for prometheus 2.13.1**

|Package Name|Version|License|
|---------|-----------|-----------|
|`cloud.google.com/go` |	v0.44.1 |	https://github.com/googleapis/google-cloud-go/blob/master/LICENSE
`contrib.go.opencensus.io/exporter/ocagent` |	v0.6.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#contrib.go.opencensus.io_exporter_ocagent-v0.6.0)
|`github.com/Azure/azure-sdk-for-go` |	v23.2.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_Azure_azure-sdk-for-go-v23.2.0)
|`github.com/Azure/go-autorest` |	v11.2.8 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_Azure_go-autorest-v11.2.8)
|`github.com/OneOfOne/xxhash` |	v1.2.5 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_OneOfOne_xxhash-v1.2.5)
|`github.com/alecthomas/units` 	|c3de453c63f4 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_alecthomas_units-c3de453c63f4)
|`github.com/armon/go-metrics` |	ec5e00d3c878 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_armon_go-metrics-ec5e00d3c878)
|`github.com/aws/aws-sdk-go` |	v1.23.12 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_aws_aws-sdk-go-v1.23.12)
|`github.com/cespare/xxhash` |	v1.1.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_cespare_xxhash-v1.1.0)
|`github.com/dgryski/go-sip13` |	25c5027a8c7b |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_dgryski_go-sip13-25c5027a8c7b)
|`github.com/edsrzf/mmap-go` |	v1.0.0 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_edsrzf_mmap-go-v1.0.0)
|`github.com/evanphx/json-patch` |	v4.5.0 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_evanphx_json-patch-v4.5.0)
|`github.com/go-kit/kit` |	v0.9.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_go-kit_kit-v0.9.0)
|`github.com/go-logfmt/logfmt` |	v0.4.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_go-logfmt_logfmt-v0.4.0)
|`github.com/go-openapi/analysis` |	v0.19.4 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_go-openapi_analysis-v0.19.4)
|`github.com/go-openapi/runtime` |	v0.19.3 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_go-openapi_runtime-v0.19.3)
|`github.com/go-openapi/strfmt` |	v0.19.2 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_go-openapi_strfmt-v0.19.2)
|`github.com/go-openapi/swag` |	v0.19.4 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_go-openapi_swag-v0.19.4)
|`github.com/gogo/protobuf` |	28a6bbf47e48 |	https://github.com/gogo/protobuf/blob/master/LICENSE
|`github.com/golang/groupcache` |	869f871628b6 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_golang_groupcache-869f871628b6)
|`github.com/golang/snappy` |	v0.0.1 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_golang_snappy-v0.0.1)
|`github.com/google/go-cmp` |	v0.3.1 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_google_go-cmp-v0.3.1)
|`github.com/google/pprof` |	34ac40c74b70 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_google_pprof-34ac40c74b70)
|`github.com/googleapis/gnostic` |	v0.3.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_googleapis_gnostic-v0.3.0)
|`github.com/gophercloud/gophercloud` |	v0.3.0 |	https://github.com/gophercloud/gophercloud/blob/master/LICENSE
|`github.com/grpc-ecosystem/grpc-gateway` |	v1.9.5 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_grpc-ecosystem_grpc-gateway-v1.9.5)
|`github.com/hashicorp/consul/api` |	v1.1.0 |	[Mozilla Public License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_hashicorp_consul_api-v1.1.0)
|`github.com/hashicorp/go-immutable-radix` |	v1.1.0 |	[Mozilla Public License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_hashicorp_go-immutable-radix-v1.1.0)
|`github.com/hashicorp/go-msgpack` |	v0.5.5 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_hashicorp_go-msgpack-v0.5.5)
|`github.com/hashicorp/go-rootcerts` |	v1.0.1 |	[Mozilla Public License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_hashicorp_go-rootcerts-v1.0.1)
|`github.com/hashicorp/golang-lru`|	v0.5.3 |	[Mozilla Public License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_hashicorp_golang-lru-v0.5.3)
|`github.com/hashicorp/memberlist` |	v0.1.4 |	[Mozilla Public License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_hashicorp_memberlist-v0.1.4)
|`github.com/hashicorp/serf` |	v0.8.3 |	[Mozilla Public License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_hashicorp_serf-v0.8.3)
|`github.com/influxdata/influxdb` |	v1.7.7 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_influxdata_influxdb-v1.7.7)
|`github.com/jpillora/backoff` |	3050d21c67d7 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_jpillora_backoff-3050d21c67d7)
|`github.com/json-iterator/go` |	v1.1.7 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_json-iterator_go-v1.1.7)
|`github.com/mailru/easyjson` |	b2ccc519800e |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_mailru_easyjson-b2ccc519800e)
|`github.com/miekg/dns` |	v1.1.15 |	https://github.com/miekg/dns/blob/master/LICENSE
|`github.com/mwitkow/go-conntrack` |	2f068394615f |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_mwitkow_go-conntrack-2f068394615f)
|`github.com/oklog/run` |	v1.0.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_oklog_run-v1.0.0)
|`github.com/oklog/ulid` |	v1.3.1 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_oklog_ulid-v1.3.1)
|`github.com/opentracing-contrib/go-stdlib` |	cf7a6c988dc9 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_opentracing-contrib_go-stdlib-cf7a6c988dc9)
|`github.com/opentracing/opentracing-go` |	v1.1.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_opentracing_opentracing-go-v1.1.0)
|`github.com/pkg/errors` |	v0.8.1 |	[BSD 2-Clause "Simplified" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_pkg_errors-v0.8.1)
|`github.com/prometheus/alertmanager` |	v0.18.0 |	https://github.com/prometheus/alertmanager/blob/master/LICENSE
|`github.com/prometheus/client_golang` |	v1.2.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_prometheus_client_golang-v1.2.0)
|`github.com/prometheus/client_model` |	14fe0d1b01d4 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_prometheus_client_model-14fe0d1b01d4)
|`github.com/prometheus/common` |	v0.7.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_prometheus_common-v0.7.0)
|`github.com/samuel/go-zookeeper` |	0ceca61e4d75 |	https://github.com/samuel/go-zookeeper/blob/master/LICENSE
|`github.com/shurcooL/httpfs` |	8d4bc4ba7749 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_shurcooL_httpfs-8d4bc4ba7749)
|`github.com/shurcooL/vfsgen` |	6a9ea43bcacd |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_shurcooL_vfsgen-6a9ea43bcacd)
|`github.com/soheilhy/cmux` |	v0.1.4 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_soheilhy_cmux-v0.1.4)
|`github.com/spaolacci/murmur3` |	v1.1.0 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_spaolacci_murmur3-v1.1.0)
|`github.com/stretchr/testify` |	v1.4.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_stretchr_testify-v1.4.0)
|`go.mongodb.org/mongo-driver` |	v1.0.4 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#go.mongodb.org_mongo-driver-v1.0.4)
|`golang.org/x/crypto` |	4def268fd1a4 |	https://github.com/golang/crypto/blob/master/LICENSE
|`golang.org/x/net` |	ca1201d0de80 |	https://github.com/golang/net/blob/master/LICENSE
|`golang.org/x/oauth2` |	0f29369cfe45 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#golang.org_x_oauth2-0f29369cfe45)
|`golang.org/x/sys` |	b09406accb47 |	https://github.com/golang/sys/blob/master/LICENSE
|`golang.org/x/sync` |	112230192c58 |	https://github.com/golang/sync/blob/master/LICENSE
|`golang.org/x/time` 	|9d24e82272b4 |	https://github.com/golang/time/blob/master/LICENSE
|`golang.org/x/tools` |	5a1a30219888 |	https://github.com/golang/tools/blob/master/LICENSE
|`google.golang.org/api` |	v0.8.0 |	https://github.com/googleapis/google-api-go-client/blob/master/LICENSE
|`google.golang.org/genproto` |	fa694d86fc64 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#google.golang.org_genproto-fa694d86fc64)
|`google.golang.org/grpc` |	v1.22.1 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#google.golang.org_grpc-v1.22.1)
|`gopkg.in/alecthomas/kingpin.v2` |	v2.2.6 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#gopkg.in_alecthomas_kingpin.v2-v2.2.6)
|`gopkg.in/check.v1` |	41f04d3bba15 |	https://github.com/go-check/check/blob/v1/LICENSE
|`gopkg.in/fsnotify/fsnotify.v1` |	v1.4.7 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#gopkg.in_fsnotify_fsnotify.v1-v1.4.7)
|`gopkg.in/inf.v0` |	v0.9.1 |	https://github.com/go-inf/inf/blob/master/LICENSE
|`gopkg.in/yaml.v2` |	v2.2.2 |	https://github.com/go-yaml/yaml/blob/v2/LICENSE
|`k8s.io/api` |	36bff7324fb7 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_api-36bff7324fb7)
|`k8s.io/apimachinery` |	423f5d784010 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_apimachinery-423f5d784010)
|`k8s.io/client-go` |	78d2af792bab |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_client-go-78d2af792bab)
|`k8s.io/klog` |	v0.4.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_klog-v0.4.0)
|`k8s.io/kube-openapi` |	5e22f3d471e6 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_kube-openapi-5e22f3d471e6)
|`k8s.io/utils` |	6c36bc71fc4a |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_utils-6c36bc71fc4a)

### 2.4 Helm

**Table 9 Software Package Licensing Information for Helm**

|Component|RPM Packages|Container Images|License Information|Dependencies
|---------|-----------|-----------|-----------|-----------|
|helm |`helm-3.1.1-1.0.1.el7.x86_64.rpm `|No container images. |[Apache 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/sw-licenses.html#comp-helm-3.1.1)|For software package dependency licensing information see Table 10, “Software Package Dependency Licensing Information for helm 3.1.1”.

**Table 10 Software Package Dependency Licensing Information for helm 3.1.1**

|Package Name|Version|License|
|---------|-----------|-----------|
|`github.com/BurntSushi/toml` |	v0.3.1 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_BurntSushi_toml-v0.3.1)
|`github.com/Masterminds/semver/v3` |	v3.0.3 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_Masterminds_semver_v3-v3.0.3)
|`github.com/Masterminds/sprig/v3` |	v3.0.2 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_Masterminds_sprig_v3-v3.0.2)
|`github.com/Masterminds/vcs` |	v1.13.1 |	https://github.com/Masterminds/vcs/blob/master/LICENSE.txt
|`github.com/asaskevich/govalidator` |	475eaeb16496 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_asaskevich_govalidator-475eaeb16496)
|`github.com/containerd/containerd` |	v1.3.2 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_containerd_containerd-v1.3.2)
|`github.com/cyphar/filepath-securejoin` |	v0.2.2 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_cyphar_filepath-securejoin-v0.2.2)
|`github.com/deislabs/oras` |	v0.8.1 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_deislabs_oras-v0.8.1)
|`github.com/docker/distribution` |	v2.7.1 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_docker_distribution-v2.7.1)
|`github.com/docker/docker` |	46ec8731fbce |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_docker_docker-46ec8731fbce)
|`github.com/docker/go-units` |	v0.4.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_docker_go-units-v0.4.0)
|`github.com/evanphx/json-patch` |	v4.5.0 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_evanphx_json-patch-v4.5.0)
|`github.com/gobwas/glob` |	v0.2.3 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_gobwas_glob-v0.2.3)
|`github.com/gofrs/flock` |	v0.7.1 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_gofrs_flock-v0.7.1)
|`github.com/gosuri/uitable` |	v0.0.4 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_gosuri_uitable-v0.0.4)
|`github.com/mattn/go-shellwords` |	v1.0.9 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_mattn_go-shellwords-v1.0.9)
|`github.com/mitchellh/copystructure` |	v1.0.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_mitchellh_copystructure-v1.0.0)
|`github.com/opencontainers/go-digest` |	v1.0.0-rc1 |	https://github.com/opencontainers/go-digest/blob/master/LICENSE
|`github.com/opencontainers/image-spec` |	v1.0.1 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_opencontainers_image-spec-v1.0.1)
|`github.com/pkg/errors` |	v0.9.1 |	[BSD 2-Clause "Simplified" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_pkg_errors-v0.9.1)
|`github.com/sirupsen/logrus` |	v1.4.2 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_sirupsen_logrus-v1.4.2)
|`github.com/spf13/cobra` |	v0.0.5 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_spf13_cobra-v0.0.5)
|`github.com/spf13/pflag` |	v1.0.5 |	[BSD 3-Clause "New" or "Revised" License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_spf13_pflag-v1.0.5)
|`github.com/stretchr/testify` |	v1.4.0 |	[MIT License](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_stretchr_testify-v1.4.0)
|`github.com/xeipuuv/gojsonschema` |	v1.1.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#github.com_xeipuuv_gojsonschema-v1.1.0)
|`golang.org/x/crypto` |	69ecbb4d6d5d |	https://github.com/golang/crypto/blob/master/LICENSE
|`k8s.io/api` 	|v0.17.2 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_api-v0.17.2)
|`k8s.io/apiextensions-apiserver` |	v0.17.2 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_apiextensions-apiserver-v0.17.2)
|`k8s.io/apimachinery` |	v0.17.2 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_apimachinery-v0.17.2)
|`k8s.io/cli-runtime` |	v0.17.2 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_cli-runtime-v0.17.2)
|`k8s.io/client-go` |	v0.17.2 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_client-go-v0.17.2)
|`k8s.io/klog` |	v1.0.0 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_klog-v1.0.0)
|`k8s.io/kubectl` |	v0.17.2 |	[Apache License 2.0](https://mydocs.no.oracle.com/staging-tree/olcne/1.2-licenses/en/ohc/html/third-licenses.html#k8s.io_kubectl-v0.17.2)
|`sigs.k8s.io/yaml` |	v1.1.0 |	https://github.com/ghodss/yaml/blob/master/LICENSE
