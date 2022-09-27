//go:build e2e
// +build e2e

package e2e

import (
	"github.com/Madongming/move-clouds-deployment/test/e2e/update"
)

var _ = fmw.Describe("from mod ingress update to nodeport", update.I2NSingleDeployment)
var _ = fmw.Describe("from mod nodeport update to ingress", update.N2ISingleDeployment)
