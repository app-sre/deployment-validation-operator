//go:build tools
// +build tools

package validations

import _ "golang.stackrox.io/kube-linter/pkg/templates/codegen"

//go:generate go run golang.stackrox.io/kube-linter/pkg/templates/codegen
