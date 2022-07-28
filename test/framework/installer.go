package framework

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Installer basic interface for installer
type Installer struct {
	Steps []*Install

	config *Config
	once   sync.Once
	logr.Logger
}

// NewInstaller inits a new installer
func NewInstaller(config *Config) *Installer {
	return &Installer{config: config, Steps: []*Install{}}
}

// WithLogger injects a logger
func (i *Installer) WithLogger(logger logr.Logger) {
	i.Logger = logger
}

func (i *Installer) init() (err error) {
	i.once.Do(func() {
		i.Steps = []*Install{}
		i.Logger = i.config.WithName("installer")

		if i.config == nil || i.config.Sub("install") == nil {
			return
		}
		installer := &Installer{}
		err = i.config.Sub("install").Unmarshal(installer)
		if err != nil {
			i.Error(err, "install unmarshal error", "config", i.config.Sub("install"))
			return
		}
		if installer.Steps != nil {
			i.Steps = installer.Steps
		}
		if len(i.Steps) > 0 {
			for _, inst := range i.Steps {
				inst.init()
				if inst.install == nil {
					continue
				}
				if inject, ok := inst.install.(LoggerInjector); ok {
					inject.WithLogger(i.Logger.WithName(inst.Name))
				}
				if inject, ok := inst.install.(WriterInjector); ok {
					inject.WithWriter(i.config.Stdout)
				}
			}
		}
	})
	return
}

// Install executes install commands
func (i *Installer) Install(clusterConfig ClusterConfig) (err error) {
	i.V(1).Info("Install start", "install", i)
	if err = i.init(); err != nil {
		return
	}
	i.V(1).Info("Will install", "len", len(i.Steps), "i", i)
	if len(i.Steps) == 0 {
		return
	}
	i.Info("Validating install steps...")
	root := field.NewPath("install")
	for idx, inst := range i.Steps {
		fld := root.Index(idx)
		if err = inst.Validate(fld); err != nil {
			i.Error(err, "install validation")
			return
		}
	}

	// validates then installs
	i.Info("Installing...")
	for _, inst := range i.Steps {
		if err = inst.Install(clusterConfig); err != nil {
			if inst.IgnoreFail {
				i.V(1).Info("installer failed but will ignore", "name", inst.Name, "error", err)
			} else {
				i.Error(err, "installer error", "name", inst.Name)
				return
			}
		}
	}
	return
}

// InstallExecuter executes a predefined install
type InstallExecuter interface {
	Install(ClusterConfig) (err error)
	Validate(fld *field.Path) field.ErrorList
}

// Install expands to support multiple install methods using specific implementations
// currently only supports running commands on the OS
// but can be easily expanded to provide other install methods like helm, kubectl apply etc.
type Install struct {
	Name       string
	IgnoreFail bool
	Command    *CommandInstall

	install InstallExecuter
}

func (c *Install) init() {
	switch {
	case c.Command != nil:
		c.install = c.Command
		if c.Command.Std == nil {
			c.Command.Std = os.Stdout
		}
	}
}

// Validate validates basic install
func (c *Install) Validate(root *field.Path) (err error) {
	errs := field.ErrorList{}
	if c.Name == "" {
		errs = append(errs, field.Invalid(root.Child("name"), c.Name, "cannot be empty"))
	}
	if c.install == nil {
		errs = append(errs, field.Invalid(root, c, `needs to define a type of installation method. Supported: [command]`))
	}

	err = errs.ToAggregate()
	return
}

// Install executes install command for initiated installer
// needs to execute init function first
func (c *Install) Install(config ClusterConfig) (err error) {
	c.init()
	if c.install == nil {
		err = fmt.Errorf("Install implementation not defined")
		return
	}
	err = c.install.Install(config)
	return
}

// CommandInstall uses commands to execute
type CommandInstall struct {
	Cmd  string
	Args []string
	Path string
	logr.Logger
	Std io.Writer
}

var _ LoggerInjector = &CommandInstall{}

// WithLogger allows logger to be injected
func (c *CommandInstall) WithLogger(log logr.Logger) {
	c.Logger = log
}

// WithWriter allows writer to injected
func (c *CommandInstall) WithWriter(wrt io.Writer) {
	c.Std = wrt
}

// Validate validates a command
func (c *CommandInstall) Validate(root *field.Path) (errs field.ErrorList) {
	errs = field.ErrorList{}
	if strings.TrimSpace(c.Cmd) == "" {
		errs = append(errs, field.Invalid(root.Child("cmd"), c.Cmd, "cannot be empty"))
	}
	if len(c.Args) > 0 {
		args := root.Child("args")
		for idx, arg := range c.Args {
			argIdx := args.Index(idx)
			if strings.TrimSpace(arg) == "" {
				errs = append(errs, field.Invalid(argIdx, arg, "cannot be empty"))
			}
		}
	}
	if c.Path == "" {
		c.Path = "."
	}
	return
}

// Install installs using specified command
func (c *CommandInstall) Install(clusterConfig ClusterConfig) (err error) {
	var workdir string
	workdir, err = os.Getwd()
	c.V(1).Info("Starting path configuration", "cfg", clusterConfig)
	if err != nil {
		c.Error(err, "error getting workdir")
		return
	}
	absPath := c.Path
	if !filepath.IsAbs(absPath) {
		if absPath, err = filepath.Abs(absPath); err != nil {
			c.Error(err, "error getting abs path")
			return
		}
	}
	// should change path to run command
	if workdir != absPath {
		// change to desired path
		if err = os.Chdir(c.Path); err != nil {
			c.Error(err, "error changing current path")
			return
		}
		// return to caller's context
		defer os.Chdir(workdir)
	}
	cmd := exec.Command(c.Cmd, c.Args...)
	cmd.Stderr = c.Std
	cmd.Stdout = c.Std
	err = cmd.Run()
	return

}

// CommandInstaller executes commands
type CommandInstaller struct {
	log.Logger
}
