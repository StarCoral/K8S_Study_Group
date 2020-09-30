// +build tools

// This package imports things required by build scripts, to force `go mod` to see them as dependencies
// See https://github.com/golang/go/issues/25922

package tools

import (
	_ "k8s.io/code-generator"
)