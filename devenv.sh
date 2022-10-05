# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

scriptHome=$(dirname ${0})

branch_version=$(grep verrazzano-development-version ${scriptHome}/.verrazzano-development-version | sed -e 's/verrazzano-development-version=//')
case $branch_version in
  1.4*|1.3*)
    GOVER=1.17.8
    ;;
  1.2*)
    GOVER=1.16.15
    ;;
  *)
    GOVER=1.19.2
    ;;
esac
unset branch_version

GOVERSION=${1:-1.19.2}
echo "Setting up go version ${GOVERSION}, GOPATH=${GOPATH}"

GOCMD=go${GOVERSION}

if [ ! -e $(which ${GOCMD}) ]; then
  echo "Installing Go version ${GOVERSION}"
  go install golang.org/dl/go${GOVERSION}@latest
else
  echo "Go version ${GOVERSION} already installed"
fi

${GOCMD} download

export GOROOT=$(${GOCMD} env GOROOT)
export PATH=${GOROOT}/bin:${PATH}

echo """

GOROOT: ${GOROOT}
GOPATH: ${GOPATH}

Go commmand: $(which go)

$(go version)
"""

unset scriptHome

