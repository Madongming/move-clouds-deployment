package framework

import (
	"fmt"

	"k8s.io/client-go/rest"
)

type ClusterConfig struct {
	Name     string       // cluster的名字，用来创建cluster使用。
	Rest     *rest.Config `json:"-"` // 比较底层的client，可以直接调用restfulapi，也可以用生成高级client，来直接访问k8s资源
	MasterIP string       // 集群的master ip，在restful api测试的时候可能需要
}

type ClusterProvider interface {
	Validate(config *Config) error
	Deploy(config *Config) (ClusterConfig, error)
	Destroy(config *Config) error
}

// 1. 定义工厂对象
type Factory struct{}

// 2. 工厂对象中提供创建不同实现了provider的对象
func (f Factory) Provider(config *Config) (ClusterProvider, error) {
	// 1. 检查配置
	if config.Viper == nil {
		return nil, fmt.Errorf("Viper is not init")
	}

	// 1.2 检查相关的配置
	if config.Sub("cluster") == nil {
		return nil, fmt.Errorf("cluster config is empty")
	}

	cluster := config.Sub("cluster")

	// 2. 判断创建k8s cluster的插件，调用插件来创建
	switch {
	case cluster.Sub("kind") != nil:
		return new(KindProvider), nil
	default:
		return nil, fmt.Errorf("Not support porvider %#v", cluster.AllSettings())
	}
}
