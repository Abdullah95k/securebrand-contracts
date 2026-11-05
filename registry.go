// Package contracts holds the SecureBrand unified contract: its registries,
// and (under gen/go) the code generated from proto/.
package contracts

import _ "embed"

// PlatformsYAML is registry/platforms.yaml.
//
//go:embed registry/platforms.yaml
var PlatformsYAML []byte

// TopicsYAML is registry/topics.yaml.
//
//go:embed registry/topics.yaml
var TopicsYAML []byte

// ExtrasYAML is registry/extras.yaml.
//
//go:embed registry/extras.yaml
var ExtrasYAML []byte
