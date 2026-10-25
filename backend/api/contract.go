// Package api holds the OpenAPI contract of the calculator service, embedded
// so the server can publish exactly the document that is checked in.
package api

import _ "embed"

// OpenAPI is the OpenAPI 3.1 description of the REST API, in YAML.
//
//go:embed openapi.yaml
var OpenAPI []byte
