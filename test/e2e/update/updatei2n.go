package update

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

func I2NSingleDeployment(ctx *framework.TestContext, f *framework.Framework) {
	var (
		ctTplPathIngress  = "update/testdata/create_ingress.yaml"
		ctTplPathNodeport = "update/testdata/create_nodeport.yaml"
		objIngress        = &unstructured.Unstructured{Object: make(map[string]interface{})}
		objNodeport       = &unstructured.Unstructured{Object: make(map[string]interface{})}
		dc                dynamic.Interface
		cs                *kubernetes.Clientset
		err               error
	)

	// Get client and singleDeployment obj
	BeforeEach(func() {
		dc = ctx.CreateDynamicClient()
		cs = ctx.CreateClientSet()
		err = framework.LoadYAMLToUnstructured(ctTplPathIngress, objIngress)
		Expect(err).Should(BeNil())
		err = framework.LoadYAMLToUnstructured(ctTplPathNodeport, objNodeport)
		Expect(err).Should(BeNil())
	})

	Context("Test singledeployment update from mod ingress to nodeport", func() {
		It("Should be create singledeployment mode ingress succuess", func() {
			gvr := schema.GroupVersionResource{
				Group:    "deployment.github.com",
				Version:  "v1",
				Resource: "singledeployments",
			}
			objIngress, err = dc.Resource(gvr).Namespace("default").Create(context.TODO(), objIngress, metav1.CreateOptions{})
			Expect(err).Should(BeNil())
		})

		It("Should be exsit singledeployment mode ingress", func() {
			By("Sleep 1 second for wait creating done")
			time.Sleep(time.Second)

			gvr := schema.GroupVersionResource{
				Group:    "deployment.github.com",
				Version:  "v1",
				Resource: "singledeployments",
			}
			_, err = dc.Resource(gvr).Namespace("default").Get(context.TODO(), objIngress.GetName(), metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})

		It("Should be exsit Deployment", func() {
			time.Sleep(time.Second)

			_, err = cs.AppsV1().Deployments("default").Get(context.TODO(), objIngress.GetName(), metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
		It("Should be exsit Service", func() {
			_, err = cs.CoreV1().Services("default").Get(context.TODO(), objIngress.GetName(), metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
		It("Should be exsit Ingress", func() {
			_, err = cs.NetworkingV1().Ingresses("default").Get(context.TODO(), objIngress.GetName(), metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})

		It("Should be update singledeployment to mode nodeport succuess", func() {
			gvr := schema.GroupVersionResource{
				Group:    "deployment.github.com",
				Version:  "v1",
				Resource: "singledeployments",
			}

			// Get objIngress update need
			objIngress, err = dc.Resource(gvr).Namespace("default").Get(context.TODO(), objIngress.GetName(), metav1.GetOptions{})
			Expect(err).Should(BeNil())

			objNodeport.SetResourceVersion(objIngress.GetResourceVersion())
			_, err = dc.Resource(gvr).Namespace("default").Update(context.TODO(), objNodeport, metav1.UpdateOptions{})
			Expect(err).Should(BeNil())
		})

		It("Should be not exsit Ingress", func() {
			By("Sleep 1 second for wait updating done")
			time.Sleep(time.Second)

			_, err = cs.NetworkingV1().Ingresses("default").Get(context.TODO(), objNodeport.GetName(), metav1.GetOptions{})
			Expect(err).ShouldNot(BeNil())
		})

		It("Should be delete success", func() {
			gvr := schema.GroupVersionResource{
				Group:    "deployment.github.com",
				Version:  "v1",
				Resource: "singledeployments",
			}
			err = dc.Resource(gvr).Namespace("default").Delete(context.TODO(), objNodeport.GetName(), metav1.DeleteOptions{})
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
			_, err = dc.Resource(gvr).Namespace("default").Get(context.TODO(), objNodeport.GetName(), metav1.GetOptions{})
			Expect(err).ShouldNot(BeNil())
		})

		It("Should be not exsit Deployment", func() {
			_, err = cs.AppsV1().Deployments("default").Get(context.TODO(), objNodeport.GetName(), metav1.GetOptions{})
			Expect(err).ShouldNot(BeNil())
		})

		It("Should be not exsit Service", func() {
			_, err = cs.CoreV1().Services("default").Get(context.TODO(), objNodeport.GetName(), metav1.GetOptions{})
			Expect(err).ShouldNot(BeNil())
		})

		It("Should be not exsit Ingress", func() {
			_, err = cs.NetworkingV1().Ingresses("default").Get(context.TODO(), objNodeport.GetName(), metav1.GetOptions{})
			Expect(err).ShouldNot(BeNil())
		})
	})
}
