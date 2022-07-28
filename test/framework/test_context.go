package framework

import (
	"context"

	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// TestContext context for a specific test case
// its name, namespace, config and other attributes
// may change between Context to make sure
// the most isolated environment for testing
type TestContext struct {
	context.Context

	Name      string
	Namespace string
	Config    *rest.Config
	MasterIP  string
}

// KubeClient generates k8s client based on the TestContext
func KubeClient(ctx *TestContext) (client kubernetes.Interface) {
	var err error
	client, err = kubernetes.NewForConfig(ctx.Config)
	Expect(err).To(Succeed(), "initiates a kubernetes client")
	return
}

// ContextFunc context function for test case
type ContextFunc func(ctx *TestContext, f *Framework)
