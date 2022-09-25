package framework

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var ConfigFileGroupResource = schema.GroupResource{
	Group:    "",
	Resource: "config",
}

type Config struct {
	*viper.Viper // 处理配置文件的工具

	Stdout io.Writer
	Stderr io.Writer
}

func NewConfig() *Config {
	return &Config{
		Viper:  viper.New(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// 从文件中加载配置到config对象中
func (c *Config) Load(configFile string) error {
	// 1. 设置文件名
	c.SetConfigName(filepath.Base(configFile))

	// 2. 设置文件目录
	c.AddConfigPath(filepath.Dir(configFile))

	// 3. 读入文件
	err := c.ReadInConfig()

	// 4. 处理错误
	if err != nil {
		// 4.1 处理后缀
		ext := filepath.Ext(configFile)
		if _, ok := err.(viper.ConfigFileNotFoundError); ok && ext != "" {
			c.SetConfigName(filepath.Base(configFile[0 : len(configFile)-len(ext)]))
			err = c.ReadInConfig()
		}
		if err != nil {
			switch err.(type) {
			case viper.ConfigFileNotFoundError:
				return errors.NewNotFound(ConfigFileGroupResource, fmt.Sprintf("config file \"%s\" not found", configFile))
			case viper.UnsupportedConfigError:
				return errors.NewBadRequest("not using a supported file format")
			default:
				return err
			}
		}
	}

	return nil
}

func (c *Config) WithWriter(std io.Writer) *Config {
	c.Stdout = std
	c.Stderr = std
	return c
}
