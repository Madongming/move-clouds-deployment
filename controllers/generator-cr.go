package controllers

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"text/template"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	deploymentv1 "github.com/Madongming/move-clouds-deployment/api/v1"
)

func parseTemplate(emfs embed.FS, templateName string, sd *deploymentv1.SingleDeployment) []byte {
	fsys, err := fs.Sub(emfs, "templates")
	if err != nil {
		fmt.Println("-----------", err)
		return []byte{}
	}
	tmpl, err := template.ParseFS(fsys, templateName)
	if err != nil {
		fmt.Println("===========", err)
		return []byte{}
	}

	b := new(bytes.Buffer)
	err = tmpl.Execute(b, sd)
	if err != nil {
		fmt.Println("+++++++++++", err)
		return []byte{}
	}

	return b.Bytes()
}

func newDeployment(emfs embed.FS, sd *deploymentv1.SingleDeployment) (*appsv1.Deployment, error) {
	deploy := new(appsv1.Deployment)
	if err := yaml.Unmarshal(parseTemplate(emfs, "deployment.yaml", sd), deploy); err != nil {
		return deploy, err
	}

	return deploy, nil
}

func newService(emfs embed.FS, sd *deploymentv1.SingleDeployment) (*corev1.Service, error) {
	service := new(corev1.Service)
	if err := yaml.Unmarshal(parseTemplate(emfs, "service.yaml", sd), service); err != nil {
		return service, err
	}

	return service, nil
}

func newIngress(emfs embed.FS, sd *deploymentv1.SingleDeployment) (*netv1.Ingress, error) {
	ingress := new(netv1.Ingress)
	if err := yaml.Unmarshal(parseTemplate(emfs, "ingress.yaml", sd), ingress); err != nil {
		return ingress, err
	}

	return ingress, nil
}
