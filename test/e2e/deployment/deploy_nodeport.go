package deployment

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/Madongming/move-clouds-deployment/test/framework"
)

func DeploymentNodeportSingleDeployment(ctx *framework.TestContext, f *framework.Framework) {
	var (
		ctTplPath = "deployment/testdata/create_nodeport.yaml"
		obj       = &unstructured.Unstructured{Object: make(map[string]interface{})}
		dc        dynamic.Interface
		cs        *kubernetes.Clientset
		err       error
	)

	// Get client and singleDeployment obj
	BeforeEach(func() {
		dc = ctx.CreateDynamicClient()
		cs = ctx.CreateClientSet()
		err = framework.LoadYAMLToUnstructured(ctTplPath, obj)
		Expect(err).Should(BeNil())
	})

	Context("Create singledeploy mod nodeport", func() {
		It("Should be created succuess", func() {
			gvr := schema.GroupVersionResource{
				Group:    "deployment.github.com",
				Version:  "v1",
				Resource: "singledeployments",
			}
			_, err := dc.Resource(gvr).Namespace("default").Create(context.TODO(), obj, metav1.CreateOptions{})
			Expect(err).Should(BeNil())
		})

		It("Should be exsit singledeployment", func() {
			By("Sleep 1 second for wait creating done")
			time.Sleep(time.Second)

			gvr := schema.GroupVersionResource{
				Group:    "deployment.github.com",
				Version:  "v1",
				Resource: "singledeployments",
			}
			_, err := dc.Resource(gvr).Namespace("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})

		It("Should be exsit Deployment", func() {
			time.Sleep(time.Second)

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

		It("Should be deleted success", func() {
			gvr := schema.GroupVersionResource{
				Group:    "deployment.github.com",
				Version:  "v1",
				Resource: "singledeployments",
			}
			err := dc.Resource(gvr).Namespace("default").Delete(context.TODO(), obj.GetName(), metav1.DeleteOptions{})
			Expect(err).Should(BeNil())
		})

		It("Should be not exsit singledeployment", func() {
			By("Sleep 1 second for wait deleting done")
			time.Sleep(time.Second)

			gvr := schema.GroupVersionResource{
				Group:    "deployment.github.com",
				Version:  "v1",
				Resource: "singledeployments",
			}
			_, err := dc.Resource(gvr).Namespace("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			Expect(err).ShouldNot(BeNil())
		})

		It("Should be not exsit Deployment", func() {
			_, err = cs.AppsV1().Deployments("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			Expect(err).ShouldNot(BeNil())
		})

		It("Should be not exsit Service", func() {
			_, err = cs.CoreV1().Services("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			Expect(err).ShouldNot(BeNil())
		})

		It("Should be not exsit Ingress", func() {
			_, err = cs.NetworkingV1().Ingresses("default").Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			Expect(err).ShouldNot(BeNil())
		})
	})
}
