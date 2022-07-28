package framework

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// var _ = viper.FlagValue

// ConfigFileGroupResource for a specific group file
var ConfigFileGroupResource = schema.GroupResource{
	Group:    "",
	Resource: "config",
}

// WriterInjector can inject a writer
type WriterInjector interface {
	WithWriter(io.Writer)
}

type LoggerInjector interface {
	WithLogger(logr.Logger)
}

// Config configuration for testing framework
type Config struct {
	*viper.Viper

	logr.Logger

	// for input output handling
	Stdout io.Writer
	Stderr io.Writer
}

// NewConfig builds new configuration instance
func NewConfig() *Config {
	return &Config{
		Viper:  viper.New(),
		Logger: zap.New(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

var _ LoggerInjector = &Config{}

// WithLogger injects logger
func (c *Config) WithLogger(log logr.Logger) {
	c.Logger = log
}

var _ WriterInjector = &Config{}

// WithWriter injects writer for stderr and stdout
func (c *Config) WithWriter(std io.Writer) {
	c.Stderr = std
	c.Stdout = std
}

// Load loads a file as configuration file
func (c *Config) Load(configFile string) (err error) {
	c.SetConfigName(filepath.Base(configFile))
	c.AddConfigPath(filepath.Dir(configFile))
	err = c.ReadInConfig()
	// usedFile := c.ConfigFileUsed()
	if err != nil {
		// If the user specified a file suffix, the Viper won't
		// find the file because it always appends its known set
		// of file suffices. Therefore try once more without
		// suffix.
		ext := filepath.Ext(configFile)
		if _, ok := err.(viper.ConfigFileNotFoundError); ok && ext != "" {
			c.SetConfigName(filepath.Base(configFile[0 : len(configFile)-len(ext)]))
			err = c.ReadInConfig()
		}
		if err != nil {
			// If a config was required, then parsing must
			// succeed. This catches syntax errors and
			// "file not found". Unfortunately error
			// messages are sometimes hard to understand,
			// so try to help the user a bit.
			switch err.(type) {
			case viper.ConfigFileNotFoundError:
				return errors.NewNotFound(ConfigFileGroupResource, fmt.Sprintf("configuration file \"%s\" not found", configFile))
			case viper.UnsupportedConfigError:
				return errors.NewBadRequest("not using a supported file format")
			default:
				// Something isn't right in the file.
				return err
			}
		}
	}
	if err == nil {
		c.init()
	}
	return nil
}

func (c *Config) init() {

	logOptions := zap.Options{
		Development: true,
		DestWriter:  c.Stdout,
		Level:       zapcore.DebugLevel,
	}
	// Initiates a zapOptions and tries to read from the viper config all the
	// other options... If error ill print it out
	if c.Sub("log") != nil {
		err := c.Sub("log").Unmarshal(&logOptions)
		if err != nil {
			// fallback to fmt
			fmt.Println("cannot unmarshal logger config:", err)
		}
	}
	c.WithLogger(zap.New(zap.UseFlagOptions(&logOptions)))
}
