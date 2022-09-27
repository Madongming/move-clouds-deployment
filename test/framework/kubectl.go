package framework

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type KubectlConfig struct {
	Stdout io.Writer
	Stderr io.Writer

	previousContext string
}

var contextNameRegex = regexp.MustCompile("[^a-zA-Z0-9-_]")

func (k *KubectlConfig) getContextName(clusterConfig ClusterConfig) string {
	return fmt.Sprintf("%s-context", contextNameRegex.ReplaceAllString(strings.ToLower(clusterConfig.Name), ""))
}

func (k *KubectlConfig) Command(cmd string, args ...string) *exec.Cmd {
	if k.Stdout == nil {
		k.Stdout = os.Stdout
	}
	if k.Stderr == nil {
		k.Stderr = os.Stderr
	}

	command := exec.Command(cmd, args...)
	command.Stdout = k.Stdout
	command.Stderr = k.Stderr
	return command
}

func (k *KubectlConfig) SetContext(clusterConfig ClusterConfig) error {
	if clusterConfig.Name == "" {
		return fmt.Errorf("clusterconfig is empty")
	}
	// 1. 获取当前的context，保存起来
	// kubectl config current-context
	cmd := k.Command("kubectl", "config", "current-context")
	currentContext := &bytes.Buffer{}
	cmd.Stdout = currentContext
	// Change
	err := cmd.Run()
	defer func() {
		if err == nil {
			k.previousContext = strings.TrimSpace(currentContext.String())
		}
	}()
	// 2. 从clusterConfig中生成新的context
	// 2.1 设置cluster
	// kubectl config set-cluster <contextName> --server <master ip> --insecure-skip-tls-verify=true
	contextName := k.getContextName(clusterConfig)
	if err = k.Command(
		"kubectl",
		"config",
		"set-cluster",
		contextName,
		"--server",
		clusterConfig.MasterIP,
		"--insecure-skip-tls-verify=true").Run(); err != nil {
		return err
	}
	// 2.2 设置授权
	if clusterConfig.Rest.BearerToken != "" {
		// kubectl config set-credentials <contextName> --token clusterConfig.Rest.BearerToken
		if err = k.Command("kubectl",
			"config",
			"set-credentials",
			contextName,
			"--token",
			clusterConfig.Rest.BearerToken).Run(); err != nil {
			return err
		}
	} else if clusterConfig.Rest.CertData != nil && clusterConfig.Rest.KeyData != nil {
		keyFile := fmt.Sprintf("/tmp/%s.key", contextName)
		certFile := fmt.Sprintf("/tmp/%s.crt", contextName)
		err = os.WriteFile(keyFile, clusterConfig.Rest.KeyData, 0644)
		if err != nil {
			return err
		}
		defer func() { _ = os.Remove(keyFile) }()
		err = os.WriteFile(certFile, clusterConfig.Rest.CertData, 0644)
		if err != nil {
			return err
		}
		defer func() { _ = os.Remove(certFile) }()
		// kubectl config set-credentials <contextName> --embed-certs=true --client-key <keyfile> --client-certificate <certfile>
		if err = k.Command(
			"kubectl",
			"config",
			"set-credentials",
			contextName,
			"--embed-certs=true",
			"--client-key",
			keyFile,
			"--client-certificate",
			certFile).Run(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Could not find credentials in config or credential method is not supported")
	}

	// 2.3 设置user
	// kubectl config set-context <contextName> --cluster=<contextName> --user=<contextName>
	if err = k.Command(
		"kubectl",
		"config",
		"set-context",
		contextName,
		fmt.Sprintf("--cluster=%s", contextName),
		fmt.Sprintf("--user=%s", contextName)).Run(); err != nil {
		return err
	}

	// 3. 切换到新的context
	// kubectl config use-context <contextName>
	if err = k.Command(
		"kubectl",
		"config",
		"use-context",
		contextName).Run(); err != nil {
		return err
	}

	return nil
}

func (k *KubectlConfig) DeleteContext(clusterConfig ClusterConfig) error {
	contextName := k.getContextName(clusterConfig)

	// 4. 还原之前的context
	if k.previousContext != "" {
		// kubectl config use-context <previousContext>
		_ = k.Command(
			"kubectl",
			"config",
			"use-context",
			k.previousContext).Run()
	}

	var err error
	// 1. 删除 cluster
	// kubectl config delete-cluster <contextName>
	if err = k.Command(
		"kubectl",
		"config",
		"delete-cluster",
		contextName).Run(); err != nil {
		return err
	}

	// 2. 删除 user(清空unset)
	// kubectl config unset users.<contextName>
	if err = k.Command(
		"kubectl",
		"config",
		"unset",
		fmt.Sprintf("users.%s", contextName)).Run(); err != nil {
		return err
	}

	// 3. 删除context
	// kubectl config delete-context <contextName>
	if err = k.Command(
		"kubectl",
		"config",
		"delete-context",
		contextName).Run(); err != nil {
		return err
	}

	return nil
}
