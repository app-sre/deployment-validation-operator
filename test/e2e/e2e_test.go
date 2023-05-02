//go:build e2e
// +build e2e

package e2e_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeploymentValidationOperator(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Deployment Validation Operator")
}
