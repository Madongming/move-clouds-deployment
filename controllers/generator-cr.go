package controllers

import (
	"bytes"
	"embed"
	"io/fs"
	"text/template"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	deploymentv1 "github.com/Madongming/move-clouds-deployment/api/v1"
)

//go:embed templates
var tpls embed.FS

func parseTemplate(templateName string, sd *deploymentv1.SingleDeployment) []byte {
	fsys, err := fs.Sub(tpls, "templates")
	if err != nil {
		return []byte{}
	}
	tmpl, err := template.ParseFS(fsys, templateName)
	if err != nil {
		return []byte{}
	}

	b := new(bytes.Buffer)
	err = tmpl.Execute(b, sd)
	if err != nil {
		return []byte{}
	}

	return b.Bytes()
}

func newDeployment(sd *deploymentv1.SingleDeployment) (*appsv1.Deployment, error) {
	deploy := new(appsv1.Deployment)
	if err := yaml.Unmarshal(parseTemplate("deployment.yaml", sd), deploy); err != nil {
		return deploy, err
	}

	return deploy, nil
}

func newService(sd *deploymentv1.SingleDeployment) (*corev1.Service, error) {
	service := new(corev1.Service)
	if err := yaml.Unmarshal(parseTemplate("service.yaml", sd), service); err != nil {
		return service, err
	}

	return service, nil
}

func newIngress(sd *deploymentv1.SingleDeployment) (*netv1.Ingress, error) {
	ingress := new(netv1.Ingress)
	if err := yaml.Unmarshal(parseTemplate("ingress.yaml", sd), ingress); err != nil {
		return ingress, err
	}

	return ingress, nil
}
