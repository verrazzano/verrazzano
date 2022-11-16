# psrctl

The PSR CLI is a tool to run and modify PSR test scenarios consisting of one or more `use cases`.

You can run a PSR scenario against a live cluster using existing scenarios such as `opensearch-s1`:
```
psrctl start -s ops-s1
```

To create new scenarios, create a `scenario.yaml` file under `manifests/scenarios` using the following convention:
```
name: opensearch-s1
ID: ops-s1
description: |
  This is a scenario that writes logs to STDOUT and gets logs from OpenSearch at a moderated rate. 
  The purpose of the scenario is to test a moderate load on both Fluend and OpenSearch by logging records.
usecases:
  - usecasePath: opensearch/writelogs.yaml
    overrideFile: writelogs.yaml
    description: write logs to STDOUT 10 times a second
```

## Building the CLI

The binary will be located in `$GOPATH/bin`.

Run `make install-cli` to build and install the `psrctl` CLI to your go path.

## Usage

Use the following syntax to run `psrctl` commands from your terminal window.
```
psrctl [command] [flags]
```

## Available Commands
| Command   | Definition                                  |
|-----------|---------------------------------------------|
| `explain` | Describe PSR scenarios that can be started  |
| `help`    | Help about any command                      |
| `list`    | List the running PSR scenarios              |
| `start`   | Start a PSR scenario                        |
| `stop`    | Stop a PSR scenario                         |
| `update`  | Update a running PSR scenario configuration |
| `version` | PSR CLI version information                 |

Run `psrctl --help` for additional usage information.