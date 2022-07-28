package framework

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/go-logr/logr"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/clientcmd"
)

// KindConfig configuration for kind Provider
type KindConfig struct {
	Name   string `json:"name"`
	Config string `json:"config"`
	Retain bool   `json:"retain"`
}

// KindProvider provisioning kind clusters
type KindProvider struct {
	logr.Logger
}

var _ ClusterProvider = &KindProvider{}

// NewKindProvider constructor for KindProvider
func NewKindProvider(logger logr.Logger) *KindProvider {
	return &KindProvider{
		Logger: logger,
	}
}

const kind = "kind"

func (k *KindProvider) getKindConfig(config *viper.Viper) (kindConfig *KindConfig, err error) {
	if config == nil {
		err = fmt.Errorf("configuration is nil for KindProvider")
		return
	}
	kindConfig = &KindConfig{}
	err = config.Unmarshal(kindConfig)
	if err == nil {
		if kindConfig.Name == "" {
			kindConfig.Name = kind
		}
	}
	return
}

// Validate config to make sure it can be used to provision kind cluster
func (k *KindProvider) Validate(config *Config) (err error) {
	kindCfg := config.Sub("cluster").Sub("kind")
	root := field.NewPath("cluster", "kind")
	if kindCfg == nil {
		err = field.Invalid(root, nil, `Config does not have kind configuration`)
		return
	}
	if kindCfg.GetString("name") == "" {
		kindCfg.Set("name", "kind")
	}

	return
}

// Deploy deploys a kind cluster using configuration
func (k *KindProvider) Deploy(config *Config) (k8sConfig ClusterConfig, err error) {
	var kindConfig *KindConfig
	if kindConfig, err = k.getKindConfig(config.Sub("cluster").Sub("kind")); err != nil {
		k.Error(err, "cannot get kind config")
		return
	}
	kubeConfigFile := ""
	if kindConfig.Retain && kindConfig.Name != "" {
		k.V(1).Info("checking if kind cluster is present", "name", kindConfig.Name)
		output := &bytes.Buffer{}
		cmd := exec.Command("kind", "get", "kubeconfig", "--name", kindConfig.Name)
		cmd.Stdout = output
		cmd.Stderr = config.Stderr
		checkErr := cmd.Run()

		if checkErr != nil {
			// most probably could not find kind
			// continue with creating cluster
			k.V(1).Info("cluster was not found, will deploy a new one", "name", kindConfig.Name)
		} else {
			// the output is th kubeconfig file
			if checkErr = ioutil.WriteFile("/tmp/kind.kubeconfig", output.Bytes(), 0644); checkErr == nil {
				k.V(1).Info("cluster was found and will be reused", "name", kindConfig.Name)
				kubeConfigFile = "/tmp/kind.kubeconfig"
			}
		}
	}
	// if no provided config file path we need to deploy a new cluster
	// and give the config file back
	if kubeConfigFile == "" {
		subCommands := []string{"create", "cluster", "--kubeconfig", "/tmp/kind.kubeconfig"}
		defer os.Remove("/tmp/kind.kubeconfig")
		if kindConfig.Config != "" {
			if err = ioutil.WriteFile("/tmp/kind.config.yaml", []byte(kindConfig.Config), 0644); err != nil {
				// error writing config file log
				k.Error(err, "error writting kind config file")
				return
			}
			defer os.Remove("/tmp/kind.config.yaml")
			subCommands = append(subCommands, "--config", "/tmp/kind.config.yaml")
		}
		if kindConfig.Name != "" {
			subCommands = append(subCommands, "--name", kindConfig.Name)
		}

		// creating kind cluster
		cmd := exec.Command("kind", subCommands...)
		cmd.Stderr = config.Stderr
		cmd.Stdout = config.Stdout

		if err = cmd.Run(); err != nil {
			k.Error(err, "kind create cluster error")
			return
		}
		kubeConfigFile = "/tmp/kind.kubeconfig"
	}

	k8sConfig.Name = kindConfig.Name
	k8sConfig.Rest, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		k.Error(err, "error generating kubeconfigdata from kind config", "config", k8sConfig.Rest)
		return
	}
	host, _ := url.Parse(k8sConfig.Rest.Host)
	if host != nil {
		k8sConfig.MasterIP = strings.Split(host.Host, ":")[0]
	}
	return
}

// Destroy destructs kind cluster
func (k *KindProvider) Destroy(config *Config) (err error) {
	var kindConfig *KindConfig
	if kindConfig, err = k.getKindConfig(config.Sub("cluster").Sub("kind")); err != nil {
		k.Error(err, "cannot get kind config")
		return
	}
	if kindConfig.Retain {
		k.Info("will retain kind cluster. Please cleanup manually using:\tkind delete cluster --name " + kindConfig.Name)
	} else {
		cmd := exec.Command("kind", "delete", "cluster", "--name", kindConfig.Name)
		cmd.Stderr = config.Stderr
		cmd.Stdout = config.Stdout
		if err = cmd.Run(); err != nil {
			k.Error(err, "kind delete cluster error")
			return
		}
	}

	return
}
