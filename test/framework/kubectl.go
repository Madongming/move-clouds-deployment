package framework

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
)

const (
	kubectl = "kubectl"
	config  = "config"
)

// KubectlConfig controls configuration for kubectl
// useful to use command line tooling
type KubectlConfig struct {
	logr.Logger
	Stdout io.Writer
	Stderr io.Writer

	previousContext string
}

// func NewKubectlConfig(logger log.Logger)

// Command starts command with std err and stdout
func (c *KubectlConfig) Command(cmd string, args ...string) *exec.Cmd {
	if c.Stderr == nil {
		c.Stderr = os.Stderr
	}
	if c.Stdout == nil {
		c.Stdout = os.Stdout
	}
	command := exec.Command(cmd, args...)
	command.Stderr = c.Stderr
	command.Stdout = c.Stdout
	return command
}

var contextNameRegex = regexp.MustCompile("[^a-zA-Z0-9-_]")

func (c *KubectlConfig) getContextName(clusterConfig ClusterConfig) (name string) {
	name = fmt.Sprintf("%s-context", contextNameRegex.ReplaceAllString(strings.ToLower(clusterConfig.Name), ""))
	return
}

// SetContext set contexts
// the main logic is arround getting the current context, store it
// create a new context using the configuration given and switch to the new context
func (c *KubectlConfig) SetContext(clusterConfig ClusterConfig) (err error) {
	// The current implemention is mainly revolved around using kubectl config commands
	// and change the context step by step.
	// Could be improved in the future the create different kubeconfig files
	// and make it load it directly instead of executing multiple commands
	contextName := c.getContextName(clusterConfig)
	c.V(1).Info("Starting kubectl configuration", "name", contextName)

	currentContext := &bytes.Buffer{}
	cmd := c.Command(kubectl, config, "current-context")
	cmd.Stdout = currentContext
	if err = cmd.Run(); err != nil {
		c.Error(err, "error getting current context")
		// return
	}
	defer func() {
		if err == nil {
			c.previousContext = strings.TrimSpace(currentContext.String())
			c.V(1).Info("Storing as previous context", "context", c.previousContext)
		}
	}()

	// setting cluster
	if err = c.Command(kubectl, config, "set-cluster", contextName, "--server", clusterConfig.Rest.Host, "--insecure-skip-tls-verify=true").
		Run(); err != nil {
		// log error
		c.Error(err, "error setting cluster context")
		return
	}
	// setting credentials
	// using berar token only needs to set credentials directly
	// if cert data needs to load to the file system then creates the credential
	// otherwise can directly load the credentials
	if clusterConfig.Rest.BearerToken != "" {
		err = c.Command(kubectl, config, "set-credentials", contextName, "--token", clusterConfig.Rest.BearerToken).Run()
	} else if clusterConfig.Rest.CertData != nil && clusterConfig.Rest.KeyData != nil {
		// save as file
		keyFile := "/tmp/" + contextName + ".key"
		certFile := "/tmp/" + contextName + ".crt"
		err = ioutil.WriteFile(keyFile, clusterConfig.Rest.KeyData, 0644)
		defer os.Remove(keyFile)
		err = ioutil.WriteFile(certFile, clusterConfig.Rest.CertData, 0644)
		defer os.Remove(certFile)

		// then use in config and embed
		err = c.Command(kubectl, config, "set-credentials", contextName, "--embed-certs=true",
			"--client-key", keyFile,
			"--client-certificate", certFile,
		).Run()
		// remove file

	} else if clusterConfig.Rest.CertFile != "" && clusterConfig.Rest.KeyFile != "" {
		// embed
		err = c.Command(kubectl, config, "set-credentials", contextName, "--embed-certs=true",
			"--client-key", clusterConfig.Rest.KeyFile,
			"--client-certificate", clusterConfig.Rest.CertFile,
		).Run()
	} else {
		err = fmt.Errorf("Could not find credentials in configuration or credentials method is not supported")
	}
	if err != nil {
		c.Error(err, "error setting credentials")
		return
	}
	if err = c.Command(kubectl, config, "set-context", contextName, "--cluster="+contextName, "--user="+contextName).Run(); err != nil {
		c.Error(err, "error setting context")
		return
	}
	if err = c.Command(kubectl, config, "use-context", contextName).Run(); err != nil {
		c.Error(err, "error using context")
		return
	}
	return
}

// DeleteContext deletes context from file
func (c *KubectlConfig) DeleteContext(clusterConfig ClusterConfig) (err error) {
	contextName := c.getContextName(clusterConfig)
	c.Command(kubectl, config, "delete-cluster", contextName).Run()
	c.Command(kubectl, config, "unset", "users."+contextName).Run()
	c.Command(kubectl, config, "delete-context", contextName).Run()

	if c.previousContext != "" {
		c.V(1).Info("Switching back context", "context", c.previousContext)
		c.Command(kubectl, config, "use-context", c.previousContext).Run()
		c.previousContext = ""
	}

	return
}
