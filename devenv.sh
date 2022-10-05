
GOVERSION=$(cat .go-version) 
echo "Setting up go version ${GOVERSION}"

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

