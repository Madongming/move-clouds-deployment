package deployment

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/Madongming/move-clouds-deployment/test/framework"
)

func DeploymentSingleDeployment(ctx *framework.TestContext, f *framework.Framework) {
	var (
		ctTplPath = "deployment/testdata/create.yaml"
		obj       = new(unstructured.Unstructured)
		dc        dynamic.Interface
		cs        *kubernetes.Clientset
		err       error
	)

	// Get client and singleDeployment obj
	Before(func() {
		dc = ctx.CreateDynamicClient()
		cs = ctx.CreateClientSet()
		err = framework.LoadYAMLToUnstructured(ctTplPath, obj)
		Expect(err).Should(BeNil())
	})

	Context("Create singledeploy mod ingress", func() {
		It("Should create succuess", func() {
			gvr := schema.GroupVersionResource{
				Group:    "deployment.github.com",
				Version:  "v1",
				Resource: "singledeployment",
			}
			resp, err := dc.Resource(gvr).Namespace("default").Create(context.TODO(), obj, metav1.CreateOptions{})
			Expect(err).Should(BeNil())
		})

		It("Should be exsit Deployment", func() {
			_, err = cs.AppsV1().Deployments("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
		It("Should be exsit Service", func() {
			_, err = cs.CoreV1().Services("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
		It("Should be exsit Ingress", func() {
			_, err = cs.NetworkingV1().Ingresses("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
	})

	Context("Create singledeploy mod nodeport", func() {
		It("Should create succuess", func() {
			gvr := schema.GroupVersionResource{
				Group:    "deployment.github.com",
				Version:  "v1",
				Resource: "singledeployment",
			}
			resp, err := dc.Resource(gvr).Namespace("default").Create(context.TODO(), obj, metav1.CreateOptions{})
			Expect(err).Should(BeNil)
		})

		It("Should be exsit Deployment", func() {
			_, err = cs.AppsV1().Deployments("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
		It("Should be exsit Service", func() {
			_, err = cs.CoreV1().Services("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
		It("Should be not exsit Ingress", func() {
			_, err = cs.NetworkingV1().Ingresses("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			Expect(err).ShouldNot(BeNil())
		})
	})
}
