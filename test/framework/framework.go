package framework

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Framework struct {
	Config        *Config
	ClusterConfig ClusterConfig

	factory  Factory
	provider ClusterProvider
	client   kubernetes.Interface

	configFile  string
	initTimeout float64
}

func NewFramework() *Framework {
	return &Framework{}
}

func (f *Framework) WithConfig(config *Config) {
	f.Config = config
}

func (f *Framework) Flags() *Framework {
	flag.StringVar(&f.configFile, "config", "config", "config file to used")
	flag.Float64Var(&f.initTimeout, "startup-timeout", 60*10, "startup timeout")
	return f
}

func (f *Framework) LoadConfig(writer io.Writer) *Framework {
	// 1. 执行获取参数
	flag.Parse()

	// 2. 创建config对象
	config := NewConfig()

	// 3. 加载文件内同到对象中
	if err := config.Load(f.configFile); err != nil {
		panic(err)
	}

	// 4. 替换stdout/stderr
	config.WithWriter(writer)

	// 5. 对象加入到Framework中
	f.WithConfig(config)

	return f
}
func (f *Framework) SynchronizedBeforeSuite(initFunc func()) *Framework {
	if initFunc == nil {
		initFunc = func() {
			// 1. 创建环境
			ginkgo.By("Deploying test environment")
			if err := f.DeployTestEnvironmnet(); err != nil {
				panic(err)
			}

			// 2. 初始化kubectl的配
			ginkgo.By("Kubectl switch context")
			kubectlConfig := KubectlConfig{
				Stdout: ginkgo.GinkgoWriter,
				Stderr: ginkgo.GinkgoWriter,
			}
			if err := kubectlConfig.SetContext(f.ClusterConfig); err != nil {
				panic(err)
			}
			defer func() {
				ginkgo.By("kubectl revertiong context")
				_ = kubectlConfig.DeleteContext(f.ClusterConfig)
			}()

			// 3. 安装依赖和我们的程序
			ginkgo.By("Preparing install steps")
			installer := NewInstaller(f.Config)
			ginkgo.By("Executing install steps")
			if err := installer.Install(f.ClusterConfig); err != nil {
				panic(err)
			}
		}
	}
	ginkgo.SynchronizedBeforeSuite(func() []byte {
		initFunc()
		return nil
	}, func(_ []byte) {}, f.initTimeout)
	return f
}

func (f *Framework) SynchronizedAfterSuite(destroyFun func()) *Framework {
	if destroyFun == nil {
		destroyFun = func() {
			// 回收测试环境
			ginkgo.By("Destroy test environment")
			if err := f.DestoryTestEnvironmnet(); err != nil {
				panic(err)
			}
		}
	}
	ginkgo.SynchronizedAfterSuite(func() {}, destroyFun, f.initTimeout)
	return f
}

func (f *Framework) MRun(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	result := m.Run()

	os.Exit(result)
}

func (f *Framework) Run(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	var r []ginkgo.Reporter
	r = append(r, reporters.NewJUnitReporter("e2e.xml"))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "e2e", r)
}

func (f *Framework) DeployTestEnvironmnet() error {
	// 1. 检查f.Config
	if f.Config == nil {
		return fmt.Errorf("Config is nil")
	}
	// 2. 创建provider
	ginkgo.By("Getting env provider")
	var err error
	if f.provider, err = f.factory.Provider(f.Config); err != nil {
		return err
	}

	// 3. 执行provider的验证方法
	ginkgo.By("Validating config for provider")
	if err = f.provider.Validate(f.Config); err != nil {
		return err
	}
	// 4. 执行provider的部署方法
	ginkgo.By("Deploying test env")
	if f.ClusterConfig, err = f.provider.Deploy(f.Config); err != nil {
		return err
	}

	// 5. 创建client
	if f.client, err = kubernetes.NewForConfig(f.ClusterConfig.Rest); err != nil {
		return err
	}

	return nil
}

func (f *Framework) DestoryTestEnvironmnet() error {
	// 1. 检查f.Config
	if f.Config == nil {
		return fmt.Errorf("Config is nil")
	}

	// 2. 检查f.provider是否创建
	if f.provider == nil {
		return fmt.Errorf("Provider is nil")
	}

	// 3. 执行provider的destroy
	ginkgo.By("Destroying test env")
	if err := f.provider.Destroy(f.Config); err != nil {
		return err
	}

	return nil
}

// 测试的入口函数
func (f *Framework) Describe(name string, ctxFun ContextFun) bool {
	// 整个函数是套在ginkgo的Describe中
	ginkgo.Describe(name, func() {
		// 1. 创建testcontext对象
		ctx, err := f.createTestContext(name, false)
		if err != nil {
			ginkgo.Fail("cannot create test context for" + name)
			return
		}

		// 2. 每次执行测试用例之前执行的内容
		ginkgo.BeforeEach(func() {
			ctx2, err := f.createTestContext(name, true)
			if err != nil {
				ginkgo.Fail("cannot create test context for name " + name + " namespace " + ctx2.Namespace)
				return
			}
			ctx.Config = ctx2.Config
			ctx.Namespace = ctx2.Namespace
			ctx.Name = ctx2.Name
		})

		// 3. 每次执行测试用例之后执行的内容
		ginkgo.AfterEach(func() {
			// 3.1 删除testcontext
			_ = f.deleteTestContext(ctx)
		})

		// 执行用户定义的测试函数
		ctxFun(ctx, f)
	})

	return true
}

var nsRegex = regexp.MustCompile("[^a-z0-9]")

func (f *Framework) createTestContext(name string, nsCreate bool) (*TestContext, error) {
	// 1. 检查f不是空
	if f == nil {
		return nil, nil
	}

	// 2. 创建testcontext对象
	ctx := &TestContext{Name: name}
	if f.ClusterConfig.Rest != nil {
		// 3. 填充里面的字段
		// 拷贝客户端到ctx
		ctx.Config = rest.CopyConfig(f.ClusterConfig.Rest)
		ctx.MasterIP = f.ClusterConfig.MasterIP
		// 4. 判断是否创建namespace
		if nsCreate {
			// 4.1 如何是，就创建
			// 4.1.1 不是声明式的创建namespace，而是使用GenerateName这种方式生成
			prefix := strings.ReplaceAll(
				strings.ReplaceAll(
					strings.ToLower(name),
					" ", "-"),
				"_", "-")
			prefix = nsRegex.ReplaceAllString(prefix, "")
			prefix = strings.ReplaceAll(prefix, "--", "-")
			if len(prefix) > 30 {
				prefix = prefix[0:30]
			}

			ns, err := f.client.CoreV1().Namespaces().Create(
				context.TODO(),
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: prefix + "-",
					},
				}, metav1.CreateOptions{})
			if err != nil {
				return nil, nil
			}
			ctx.Namespace = ns.GetName()
		}
	}

	// 5.... 其他操作，如创建sa/secret等

	// change
	return ctx, nil
}

func (f *Framework) deleteTestContext(ctx *TestContext) error {
	// *删除createTextContext中创建的资源
	// 删除namespace
	if err := f.client.CoreV1().Namespaces().Delete(context.TODO(), ctx.Namespace, metav1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}
