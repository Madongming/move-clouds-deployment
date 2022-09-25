package framework

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// 1. 执行i := NewInstaller(config)
// 2. 执行i.Install(clusterConfig)

type Installer struct {
	Steps []*Install

	config *Config
	once   sync.Once
	stdout io.Writer
	stderr io.Writer
}

func NewInstaller(config *Config) *Installer {
	return &Installer{
		stdout: config.Stdout,
		stderr: config.Stderr,
	}
}

func (i *Installer) init() (err error) {
	// 只能执行一次
	i.once.Do(func() {
		i.Steps = []*Install{}

		// 判断相关的配置不为空
		if i.config == nil || i.config.Sub("install") == nil {
			err = fmt.Errorf("config format error")
			return
		}

		installer := new(Installer)
		if err = i.config.Sub("install").Unmarshal(installer); err != nil {
			return
		}

		if installer.Steps != nil {
			i.Steps = installer.Steps
		}
	})
	return
}

func (i *Installer) Install(clusterConfig ClusterConfig) error {
	// 1. 执行初始化
	if err := i.init(); err != nil {
		return err
	}

	// 2. 判断steps中是否有任务
	if len(i.Steps) == 0 {
		return nil
	}

	// 3. 遍历steps，执行每一个任务的validata函数
	root := field.NewPath("install")
	for index, inst := range i.Steps {
		fld := root.Index(index)
		if err := inst.validate(fld); err != nil {
			return err
		}
	}

	// 4. 遍历steps，执行每一个任务的install函数
	for _, inst := range i.Steps {
		if err := inst.install(clusterConfig); err != nil {
			return err
		}
	}

	// 5. 设置标准输入输出
	for _, inst := range i.Steps {
		inst.stdout = i.stdout
		inst.stderr = i.stderr
	}

	return nil
}

type Install struct {
	Name       string
	IngoreFail bool
	Cmd        string
	Args       []string
	Path       string

	stdout io.Writer
	stderr io.Writer
}

func (i *Install) validate(root *field.Path) error {
	errs := field.ErrorList{}
	// 1. 验证 name 字段
	if strings.TrimSpace(i.Name) == "" {
		errs = append(errs, field.Invalid(root.Child("name"), i.Name, "Cannot be empty"))
	}
	// 2. 验证 cmd 字段
	if strings.TrimSpace(i.Cmd) == "" {
		errs = append(errs, field.Invalid(root.Child("cmd"), i.Cmd, "Cannot be empty"))
	}

	// 3. 验证path，如果为空，给出默认值 "."
	if strings.TrimSpace(i.Path) == "" {
		i.Path = "."
	}

	// 聚合错误
	err := errs.ToAggregate()

	return err
}

func (i *Install) install(config ClusterConfig) error {
	// 1. 获取当前路径
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	absPath := i.Path
	if !filepath.IsAbs(absPath) {
		if absPath, err = filepath.Abs(absPath); err != nil {
			return err
		}
	}

	// 2. 对比设置路径和当前路径，如果不同，则执行切换
	if currentDir != absPath {
		if err = os.Chdir(absPath); err != nil {
			return err
		}
	}

	// 3. 退出此函数前，切换回之前的目录
	defer func() { _ = os.Chdir(currentDir) }()

	// 4. 执行命令
	cmd := exec.Command(i.Cmd, i.Args...)
	cmd.Stdout = i.stdout
	cmd.Stderr = i.stderr

	return cmd.Run()
}
