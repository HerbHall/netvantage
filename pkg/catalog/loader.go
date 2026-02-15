package catalog

import (
	_ "embed"
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed catalog.yaml
var catalogRawData []byte

// catalogFile is the top-level structure of the embedded YAML.
type catalogFile struct {
	Entries []CatalogEntry `yaml:"entries"`
}

// Catalog provides lazy-loaded access to the embedded tool catalog.
type Catalog struct {
	once    sync.Once
	entries []CatalogEntry
	err     error
}

// NewCatalog creates a new Catalog that will parse the embedded YAML on first access.
func NewCatalog() *Catalog {
	return &Catalog{}
}

// Entries returns a copy of all catalog entries.
func (c *Catalog) Entries() ([]CatalogEntry, error) {
	c.once.Do(c.load)
	if c.err != nil {
		return nil, c.err
	}
	cp := make([]CatalogEntry, len(c.entries))
	copy(cp, c.entries)
	return cp, nil
}

// load parses the embedded YAML catalog data.
func (c *Catalog) load() {
	var f catalogFile
	if err := yaml.Unmarshal(catalogRawData, &f); err != nil {
		c.err = fmt.Errorf("catalog: parse yaml: %w", err)
		return
	}
	c.entries = f.Entries
}
