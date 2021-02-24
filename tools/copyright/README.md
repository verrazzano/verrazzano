# Copyright Scanner

This tool enforces that all required files have the necessary copyright and license 
statements near the beginning of the file.

## Usage

```shell
go run copyright.go [options] path [path ... ]

Options:
--enforce-current   Enforce that files provided to the tool have the current year in the copyright
--verbose           Verbose output
```

## Running the Scanner

The tool can be run simply by invoking it with a list of files or directory paths.

For example, to check all files in the Verrazzano repo, you can run it as follows:

```shell
$ cd ${VERRAZZANO_ROOT}
$ go run copyright.go .
Files to ignore: [platform-operator/scripts/install/config/00-crds.patch platform-operator/helm_config/charts/verrazzano/NOTES.txt platform-operator/helm_config/charts/verrazzano-application-operator/NOTES.txt LICENSES-OLCNE.pdf application-operator/deploy/application-operator.txt tests/e2e/config/scripts/terraform/cluster/required-env-vars tests/e2e/config/scripts/looping-test/types.txt .DS_Store]
Directories to ignore: [platform-operator/thirdparty .idea]

Copyright scanning target .

Results of scan:
	Files analyzed: 547
	Files with error: 0
	Files skipped: 50
	Directories skipped: 9
```

Adding `--verbose` will spit out more details.  For example:

```shell
$ go run tools/copyright/copyright.go --verbose platform-operator/scripts 
Files to ignore: [platform-operator/scripts/install/config/00-crds.patch platform-operator/helm_config/charts/verrazzano/NOTES.txt platform-operator/helm_config/charts/verrazzano-application-operator/NOTES.txt LICENSES-OLCNE.pdf application-operator/deploy/application-operator.txt tests/e2e/config/scripts/terraform/cluster/required-env-vars tests/e2e/config/scripts/looping-test/types.txt]
Directories to ignore: [platform-operator/thirdparty]

Copyright scanning target platform-operator/scripts
Scanning platform-operator/scripts/install/1-install-istio.sh
Scanning platform-operator/scripts/install/2-install-system-components.sh
Scanning platform-operator/scripts/install/3-install-verrazzano.sh
Scanning platform-operator/scripts/install/4-install-keycloak.sh
Scanning platform-operator/scripts/install/common.sh
Skipping file platform-operator/scripts/install/config/00-crds.patch
Skipping file platform-operator/scripts/install/config/config_defaults.json
Skipping file platform-operator/scripts/install/config/config_kind.json
Skipping file platform-operator/scripts/install/config/config_oci.json
Skipping file platform-operator/scripts/install/config/config_olcne.json
Scanning platform-operator/scripts/install/config/coredns-template.yaml
Scanning platform-operator/scripts/install/config/istio_intermediate_ca_config.txt
Scanning platform-operator/scripts/install/config/istio_root_ca_config.txt
Skipping file platform-operator/scripts/install/config/keycloak.json
Scanning platform-operator/scripts/install/config/verrazzano_admission_controller_ca_config.txt
Scanning platform-operator/scripts/install/config/verrazzano_admission_controller_cert_config.txt
Scanning platform-operator/scripts/install/config.sh
Scanning platform-operator/scripts/install/create_oci_config_secret.sh
Scanning platform-operator/scripts/install/install-oke.sh
Scanning platform-operator/scripts/install/k8s-dump-objects.sh
Scanning platform-operator/scripts/install/logging.sh
Skipping file platform-operator/scripts/uninstall/README.md
Skipping file platform-operator/scripts/uninstall/build/logs/uninstall-verrazzano.sh.log
Scanning platform-operator/scripts/uninstall/uninstall-steps/0-uninstall-applications.sh
Scanning platform-operator/scripts/uninstall/uninstall-steps/1-uninstall-istio.sh
Scanning platform-operator/scripts/uninstall/uninstall-steps/2-uninstall-system-components.sh
Scanning platform-operator/scripts/uninstall/uninstall-steps/3-uninstall-verrazzano.sh
Scanning platform-operator/scripts/uninstall/uninstall-steps/4-uninstall-keycloak.sh
Skipping file platform-operator/scripts/uninstall/uninstall-steps/build/logs/1-uninstall-istio.sh.log
Skipping file platform-operator/scripts/uninstall/uninstall-steps/build/logs/2-uninstall-system-components.sh.log
Skipping file platform-operator/scripts/uninstall/uninstall-steps/build/logs/3-uninstall-verrazzano.sh.log
Skipping file platform-operator/scripts/uninstall/uninstall-steps/build/logs/4-uninstall-keycloak.sh.log
Scanning platform-operator/scripts/uninstall/uninstall-utils.sh
Scanning platform-operator/scripts/uninstall/uninstall-verrazzano.sh

Results of scan:
	Files analyzed: 22
	Files with error: 0
	Files skipped: 12
	Directories skipped: 0
```
## Scanning Locally Modified Files

While doing local development, in addition to adding copyright/license statements to new files, the copyright statements 
in existing files may need to be updated to add the current year.  

You can use the scanner to check if the files modified locally have the correct copyright/license information,
including the current year, using the `--enforce-current` option.

The following example uses the `git status --short` command to obtain a set of locally modified files and validate the
copyright/licsense information:

```shell
$ go run tools/copyright/copyright.go --verbose --enforce-current  $(git status --short | cut -c 4-)
Enforcing current year in copyright string
Files to ignore: [platform-operator/scripts/install/config/00-crds.patch platform-operator/helm_config/charts/verrazzano/NOTES.txt platform-operator/helm_config/charts/verrazzano-application-operator/NOTES.txt LICENSES-OLCNE.pdf application-operator/deploy/application-operator.txt tests/e2e/config/scripts/terraform/cluster/required-env-vars tests/e2e/config/scripts/looping-test/types.txt .DS_Store]
Directories to ignore: [platform-operator/thirdparty .idea]

Copyright scanning target ignore_copyright_check.txt
Copyright scanning target tools/copyright/README.md
Copyright scanning target .DS_Store
Copyright scanning target platform-operator/run-vpo.sh

Results of scan:
	Files analyzed: 2
	Files with error: 0
	Files skipped: 2
	Directories skipped: 0
```

## Scanning Files Changed in a Branch

In combination with Git, you can use the tool to scan for files modified between branches.

The following example compares the current working branch against master to get the set of modified files and feeds it
to the scanner, and also checks for the current year in the copyright:

```shell
$ go run tools/copyright/copyright.go --enforce-current $(git diff --name-only origin/master) 
Enforcing current year in copyright string
Files to ignore: [platform-operator/scripts/install/config/00-crds.patch platform-operator/helm_config/charts/verrazzano/NOTES.txt platform-operator/helm_config/charts/verrazzano-application-operator/NOTES.txt LICENSES-OLCNE.pdf application-operator/deploy/application-operator.txt tests/e2e/config/scripts/terraform/cluster/required-env-vars tests/e2e/config/scripts/looping-test/types.txt .DS_Store]
Directories to ignore: [platform-operator/thirdparty .idea]

Copyright scanning target Jenkinsfile
Copyright scanning target Makefile
Copyright scanning target tools/copyright/README.md
Copyright scanning target tools/copyright/copyright.go

Results of scan:
	Files analyzed: 3
	Files with error: 0
	Files skipped: 1
	Directories skipped: 0
```
