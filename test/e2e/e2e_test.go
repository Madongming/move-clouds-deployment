//go:build e2e
// +build e2e

package e2e

import (
	"testing"

	"github.com/onsi/ginkgo"

	"github.com/Madongming/move-clouds-deployment/test/framework"
)

var fmw = framework.NewFramework()

func TestMain(m *testing.M) {
	fmw.Flags().
		LoadConfig(ginkgo.GinkgoWriter).
		SynchronizedBeforeSuite(nil).
		SynchronizedAfterSuite(nil).
		MRun(m)
}

func TestE2E(t *testing.T) {
	// start step to run e2e
	fmw.Run(t)
}
