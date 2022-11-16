# psrctl

The PSR CLI is a tool to run and modify PSR test scenarios consisting of one or more `use cases`.

You can run a PSR scenario against a live cluster using existing scenarios such as `opensearch-s1`:
```
psrctl start -s ops-s1
```

To create new scenarios, create a `scenario.yaml` file under `manifests/scenarios/myscenario` using the following convention:
```
name: myscenario-1
ID: ops-myscenario-1
description: |
  This is a new scenario. It runs the use case opensearch/writelogs.yaml.
usecases:
  - usecasePath: opensearch/writelogs.yaml
    overrideFile: writelogs.yaml
    description: write logs to STDOUT 10 times a second
```

To start the above scenario, run:
```
psrctl start -d manifests/scenarios/myscenario -s ops-myscenario-1
```
The flag `-d` allows you to specify a scenario directory that is not compiled into the `psrctl` binary, such as the one created above.

For newly created scenarios, a `usecase-overrides` directory must be provided with each override values for each `use case`.
See the file structure of `manifests/scenarios/opensearch/s1` as an example.

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