//+build tools

// Package tools declares the tool dependencies of this project.
//
// This approach is described in https://go.indeed.com/gotools
// and https://github.com/golang/go/issues/25922#issuecomment-451123151.
package tools

import (
	_ "github.com/vektra/mockery/v2/cmd"

	_ "oss.indeed.com/go/go-groups"
)
