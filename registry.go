// Package contracts holds the SecureBrand unified contract: its registries,
// and (under gen/go) the code generated from proto/.
package contracts

import _ "embed"

// Version is the contract version. Producers put it in every record's
// "sb-contract" header.
const Version = "1.2.0"

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
