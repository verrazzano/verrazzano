package catalog

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const catalogPath = "./catalog.yaml"

func TestNewCatalog(t *testing.T) {
	catalog, err := NewCatalog(catalogPath)
	assert.NoError(t, err)
	assert.NotNil(t, catalog)
	assert.Equal(t, len(catalog.versionMap), len(catalog.Modules))
}
