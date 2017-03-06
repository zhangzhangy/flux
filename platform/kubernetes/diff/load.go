package diff

import (
	"errors"
)

// LoadManifests takes a path to a directory or file, and creates an
// object set based on the file(s) therein. Resources are named
// according to the file content, rather than the file name of
// directory structure.
func Load(path string) (ObjectSet, error) {
	return nil, errors.New("not implemented")
}

// ParseManifests takes a dump of config (a multidoc YAML) and
// constructs an object set from the resources represented therein.
func ParseMultidoc(multidoc string) (ObjectSet, error) {
	return nil, errors.New("not implemented")
}
