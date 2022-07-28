package framework

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	tenSeconds float64 = 10
)

// Framework basic testing framework
type Framework struct {
	Config *Config
	logr.Logger
	factory       Factory
	ClusterConfig ClusterConfig
	client        kubernetes.Interface
	provider      ClusterProvider

	// flags
	configFile       string
	initilizeTimeout float64
}

// NewFramework constructor
func NewFramework() *Framework {
	return &Framework{
		Logger: zap.New(),
	}
}

// Flags initializes basic flags
func (f *Framework) Flags() *Framework {
	flag.StringVar(&f.configFile, "config", "config", "The name of a config file (https://github.com/spf13/viper#what-is-viper). All e2e command line parameters can also be configured in such a file. May contain a path and may or may not contain the file suffix. The default is to look for an optional file with `e2e` as base name. If a file is specified explicitly, it must be present.")
	flag.Float64Var(&f.initilizeTimeout, "startup-timeout", 60*10, `timeout to startup the whole framework and prepare environment`)

	return f
}

// LoadConfig loads configuration into framework
func (f *Framework) LoadConfig(writer io.Writer) *Framework {
	flag.Parse()

	config := NewConfig()
	config.init()

	config.WithWriter(writer)
	err := config.Load(f.configFile)
	if err != nil {
		panic(err)
	}
	f.WithConfig(config)

	return f
}

// MRun main testing.M run
func (f *Framework) MRun(m *testing.M) {

	rand.Seed(time.Now().UnixNano())
	result := m.Run()

	os.Exit(result)
}

// WithConfig adds a config to framework
func (f *Framework) WithConfig(config *Config) *Framework {
	f.Logger = config.Logger
	f.Config = config
	return f
}

// SynchronizedBeforeSuite basic before suite initialization
func (f *Framework) SynchronizedBeforeSuite(initFunc func()) *Framework {
	if initFunc == nil {
		initFunc = func() {
			logger := f.WithName("BeforeSuite")

			ginkgo.By("deploying test environment")
			err := f.DeployTestEnvironment()
			if err != nil {
				logger.Error(err, "Cannot deploy test environment")
				panic(err)
			}

			// switches kubectl context to the new config
			// to make sure new runned commands run in the cluster
			ginkgo.By("kubectl switching context")
			kubectlCfg := &KubectlConfig{
				Logger: logger.WithName("kubectl"),
				Stderr: ginkgo.GinkgoWriter,
				Stdout: ginkgo.GinkgoWriter,
			}
			if err = kubectlCfg.SetContext(f.ClusterConfig); err != nil {
				logger.Error(err, "kubectl context change failed")
				panic(err)
			}
			defer func() {
				ginkgo.By("kubectl reverting context")
				kubectlCfg.DeleteContext(f.ClusterConfig)
			}()

			// install necessary software
			ginkgo.By("preparing install steps")
			installer := NewInstaller(f.Config)
			installer.WithLogger(logger.WithName("installer"))
			ginkgo.By("executing install steps")
			err = installer.Install(f.ClusterConfig)

			if err != nil {
				logger.Error(err, "install failed")
				panic(err)
			}
		}
	}
	ginkgo.SynchronizedBeforeSuite(func() []byte {
		initFunc()
		return nil
	}, func(_ []byte) {
		// no-op for now
	}, f.initilizeTimeout)

	return f
}

// SynchronizedAfterSuite destroys the whole environment
func (f *Framework) SynchronizedAfterSuite(destroyFunc func()) *Framework {
	if destroyFunc == nil {
		destroyFunc = func() {
			logger := f.WithName("AfterSuite")
			logger.Info("SynchronizedAfterSuite")
			ginkgo.By("destroy test environment")
			err := f.DestroyTestEnvironment()
			if err != nil {
				logger.Error(err, "destroy test environment")
			}
		}
	}
	ginkgo.SynchronizedAfterSuite(func() {}, destroyFunc, f.initilizeTimeout)
	return f
}

// Run start tests
func (f *Framework) Run(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	var r []ginkgo.Reporter
	r = append(r, reporters.NewJUnitReporter("e2e.xml"))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "e2e", r)
}

// DeployTestEnvironment deploy test environment
func (f *Framework) DeployTestEnvironment() (err error) {
	if f.Config == nil {
		err = fmt.Errorf("Config is nil, should be configured")
		return
	}
	ginkgo.By("getting env provider")

	if f.provider, err = f.factory.Provider(f.Config); err != nil {
		f.Error(err, "error getting cluster provider")
		return
	}

	ginkgo.By("validating configuration for provider")
	if err = f.provider.Validate(f.Config); err != nil {
		f.Error(err, "validation error")
		return
	}

	ginkgo.By("deploying test environment")
	if f.ClusterConfig, err = f.provider.Deploy(f.Config); err != nil {
		f.Error(err, "deployment error")
		return
	}
	if f.client, err = kubernetes.NewForConfig(f.ClusterConfig.Rest); err != nil {
		f.Error(err, "kubernetes client error")
	}
	return
}

// DestroyTestEnvironment destroy environment
func (f *Framework) DestroyTestEnvironment() (err error) {
	if f.Config == nil {
		err = fmt.Errorf("Config is nil, should be configured")
		return
	}
	if f.provider == nil {
		err = fmt.Errorf("Provider is nil, should have deployed before")
		return
	}
	ginkgo.By("destroying test environment")
	if err = f.provider.Destroy(f.Config); err != nil {
		f.Error(err, "destroy environment error")
	}
	return
}

// Describe wraps ginkgo's Describe function to provider a testing context
// and make testing easier
// In this Describe function the main purpose is to
// 1. create a working namespace for the test based on the test name
// 2. initializes a rest.Config with a working client for access
func (f *Framework) Describe(name string, ctxFunc ContextFunc) bool {
	ginkgo.Describe(name, func() {
		// initializes the basic test context
		ctx, err := f.createTestContext(name, false)
		if err != nil {
			ginkgo.Fail("cannot create test context for " + name)
			return
		}

		// creates the namespace, sa and other things
		ginkgo.BeforeEach(func() {
			ctx2, err := f.createTestContext(name, true)
			if err != nil {
				ginkgo.Fail("cannot create test context for " + name + " namespace " + ctx.Namespace)
				return
			}
			ctx.Config = ctx2.Config
			ctx.Namespace = ctx2.Namespace
			ctx.Name = ctx2.Name
		})

		// Deletes the whole context just after the test
		ginkgo.AfterEach(func() {
			f.deleteTestContext(ctx)
		})

		// calls the real context function provided by the user
		ctxFunc(ctx, f)
	})
	return true
}

// PDescribe wraps ginkgo's PDescribe function
func (f *Framework) PDescribe(name string, ctxFunc ContextFunc) bool {
	ginkgo.PDescribe(name, func() {
		ctxFunc(&TestContext{}, f)
	})
	return true
}

var nsRegex = regexp.MustCompile("[^a-z0-9-]")

func (f *Framework) createTestContext(name string, createNs bool) (ctx *TestContext, err error) {
	if f == nil {
		return
	}
	// create context
	ctx = &TestContext{Name: name}
	if f.ClusterConfig.Rest != nil {
		// copy client to context
		ctx.Config = rest.CopyConfig(f.ClusterConfig.Rest)
		ctx.MasterIP = f.ClusterConfig.MasterIP

		if createNs {
			// create namespace
			prefix := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(name), " ", "-"), "_", "-")
			prefix = nsRegex.ReplaceAllString(prefix, "")
			prefix = strings.ReplaceAll(prefix, "--", "-")
			if len(prefix) > 30 {
				prefix = prefix[0:30]
			}

			var ns *corev1.Namespace
			if ns, err = f.client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: prefix + "-",
				},
			}, metav1.CreateOptions{}); err != nil {
				f.Info("Test context create namespace error", "err", err)
				return
			}

			ctx.Namespace = ns.GetName()
		}
	}
	return
}

func (f *Framework) deleteTestContext(ctx *TestContext) (err error) {
	// delete namespace
	err = f.client.CoreV1().Namespaces().Delete(ctx, ctx.Namespace, metav1.DeleteOptions{})
	if err != nil && errors.IsNotFound(err) {
		// does not exist, just return
		f.Info("namespace was not found", "error", err, "name", ctx.Namespace)
		return
	} else if err != nil {
		// some other error other than internal error
		// we should retry?
		f.Info("error deleting namespace for test context", "name", ctx.Namespace, "error", err)
		return
	}

	return
}
