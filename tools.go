//go:build tools

// Package tools tracks go tooling that is needed for working with this repository.
package tools

// Follow this best practice when updating this file: https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
