//go:build e2e
// +build e2e

package e2e

import (
	"github.com/Madongming/move-clouds-deployment/test/e2e/deployment"
)

var _ = fmw.Describe("deployment ingress server", deployment.DeploymentIngressSingleDeployment)
var _ = fmw.Describe("deployment nodeport server", deployment.DeploymentNodeportSingleDeployment)
