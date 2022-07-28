package framework

import (
	"fmt"

	"k8s.io/client-go/rest"
)

// ClusterProvider interface to provide a cluster
type ClusterProvider interface {
	Validate(config *Config) (err error)
	Deploy(config *Config) (ClusterConfig, error)
	Destroy(config *Config) (err error)
}

// ClusterConfig configuration for kubernetes
type ClusterConfig struct {
	Name     string
	Rest     *rest.Config `json:"-"`
	MasterIP string
}

// Factory basic multiple usage factory
// for cluster provider, installer etc
type Factory struct {
}

func (f Factory) basicCheck(config *Config) (err error) {
	if config.Viper == nil {
		err = fmt.Errorf("needs to have a viper configuration in config")
	}
	return
}

// Provider provides a ClusterProvider based on config
func (f Factory) Provider(config *Config) (prov ClusterProvider, err error) {
	if err = f.basicCheck(config); err != nil {
		return
	}
	if config.Sub("cluster") == nil {
		err = fmt.Errorf("there is no cluster configuration in configuration file")
		return
	}
	cluster := config.Sub("cluster")
	// TODO: change to a more coding friendly method
	switch {
	case cluster.Sub("kind") != nil:
		prov = NewKindProvider(config.Logger.WithName("kind"))
	default:
		err = fmt.Errorf("selected cluster configuration \"%#v\" not supported", cluster.AllSettings())
	}
	return
}

// Installer installing stuff
func (f Factory) Installer(config *Config) (inst *Installer, err error) {
	if err = f.basicCheck(config); err != nil {
		return
	}
	inst = NewInstaller(config)
	if config.Sub("install") == nil {
		// return no-op installer
		return
	}
	return
}
