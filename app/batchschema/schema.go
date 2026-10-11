// Package batchschema holds the canonical public contracts for mentor batch inputs.
// CI checks their supported structure and constraints against the Go parser and
// validators. The exporter publishes these embedded files alongside class and
// scoreboard exports for AI agents.
package batchschema

import "embed"

//go:embed *.json
var FS embed.FS
