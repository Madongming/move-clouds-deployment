package framework

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// 1. 定义一个测试函数，他的函数签名符合 func(ctx *TestContext, f *Framework)
// 2. 我们定义一个入口函数，在函数中我们可以做以下事情：
//    2.1 创建namespace
//    2.2 创建sa/secret
//    2.3 等等
//    可以把这些结果的状态放在ctx中，供测试中使用
// 3. 我们执行用户定义的测试函数，并把ctx和包含用一些扩展方法的fmw对象一起传入这个函数中

type TestContext struct {
	Name      string
	Namespace string
	Config    *rest.Config
	MasterIP  string
}

type ContextFun func(ctx *TestContext, f *Framework)

func (tc *TestContext) CreateDynamicClient() dynamic.Interface {
	By("Create Dynamic Client")
	c, err := dynamic.NewForConfig(tc.Config)
	Expect(err).Should(BeNil())
	return c
}

func (tc *TestContext) CreateClientSet() *kubernetes.Clientset {
	By("Create ClientSet")
	c, err := kubernetes.NewForConfig(tc.Config)
	Expect(err).Should(BeNil())
	return c
}
