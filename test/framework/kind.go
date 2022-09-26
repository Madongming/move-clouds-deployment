package framework

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"

	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/clientcmd"
)

type KindConfig struct {
	Name   string `json:"name"`
	Config string `json:"config"`
	Retain bool   `json:"retain"`
}

type KindProvider struct {
}

var _ ClusterProvider = &KindProvider{}

func (k *KindProvider) Validate(config *Config) error {
	// 1. 获取配置
	kindConfig := config.Sub("cluster").Sub("kind")
	root := field.NewPath("cluster", "kind")

	if kindConfig == nil {
		return field.Invalid(root, nil, "Config does not have kind configuration")
	}

	// 2. 检查必要项
	if kindConfig.GetString("name") == "" {
		// 3. 设置默认项
		kindConfig.Set("name", "e2e")
	}

	return nil
}

func (k *KindProvider) Deploy(config *Config) (ClusterConfig, error) {
	// 1. 获取配置
	kindConfig, err := getKindConfig(config.Sub("cluster").Sub("kind"))
	if err != nil {
		return ClusterConfig{}, err
	}

	var kubeConfigFile string
	if kindConfig.Retain && kindConfig.Name != "" {
		// 2. 判断是否存在cluster
		output := &bytes.Buffer{}
		// 获取kubeconfig文件内容
		// kind get kubeconfig --name e2e
		cmd := exec.Command("kind", "get", "kubeconfig", "--name", kindConfig.Name)
		cmd.Stdout = output
		cmd.Stderr = config.Stderr
		checkErr := cmd.Run()
		if checkErr == nil {
			// 3. 存在就生成k8s链接的配置文件
			if checkErr = os.WriteFile("/tmp/kind.kubeconfig", output.Bytes(), 0644); checkErr != nil {
				return ClusterConfig{}, checkErr
			}
			kubeConfigFile = "/tmp/kind.kubeconfig"
		}
	}

	// 4. 如果不存在就创建集群
	if kubeConfigFile == "" {
		// 创建命令 kind create cluster --config /tmp/kind.config.yaml --kubeconfig /tmp/kind.kubeconfig --name e2e
		subCommand := []string{"create", "cluster", "--kubeconfig", "/tmp/kind.kubeconfig"}
		defer func() { _ = os.Remove("/tmp/kind.kubeconfig") }()
		if kindConfig.Config != "" {
			// 创建 命令参数中 config 需要的文件
			if err := os.WriteFile("/tmp/kind.config.yaml", []byte(kindConfig.Config), 0644); err != nil {
				return ClusterConfig{}, err
			}
			defer func() { _ = os.Remove("/tmp/kind.config.yaml") }()
			subCommand = append(subCommand, "--config", "/tmp/kind.config.yaml")
		}
		subCommand = append(subCommand, "--name", kindConfig.Name)

		cmd := exec.Command("kind", subCommand...)
		cmd.Stdout = config.Stdout
		cmd.Stderr = config.Stderr
		if err := cmd.Run(); err != nil {
			return ClusterConfig{}, err
		}
		kubeConfigFile = "/tmp/kind.kubeconfig"
	}

	// 5. 创建配置对象，其中包含访问k8s的client
	k8sConfig := ClusterConfig{
		Name: kindConfig.Name,
	}

	if k8sConfig.Rest, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile); err != nil {
		return ClusterConfig{}, err
	}

	host, _ := url.Parse(k8sConfig.Rest.Host)
	if host != nil {
		k8sConfig.MasterIP = fmt.Sprintf("https://%s", host.Host)
	}

	return k8sConfig, nil
}

func (k *KindProvider) Destroy(config *Config) error {
	// 1. 获取配置
	kindConfig, err := getKindConfig(config.Sub("cluster").Sub("kind"))
	if err != nil {
		return err
	}
	// 2. 判断是否要保留
	if kindConfig.Retain {
		return nil
	}

	// 3. 不保留就删除
	// kind delete cluster --name e2e
	cmd := exec.Command("kind", "delete", "cluster", "--name", kindConfig.Name)
	cmd.Stdout = config.Stdout
	cmd.Stderr = config.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func getKindConfig(config *viper.Viper) (*KindConfig, error) {
	if config == nil {
		return nil, fmt.Errorf("Config is nol for KindProvider")
	}
	kindConfig := new(KindConfig)
	if err := config.Unmarshal(kindConfig); err != nil {
		return nil, err
	}

	if kindConfig.Name == "" {
		kindConfig.Name = "e2e"
	}

	return kindConfig, nil
}
