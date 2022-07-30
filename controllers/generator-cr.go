package controllers

import (
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation/field"

	deploymentv1 "github.com/Madongming/move-clouds-deployment/api/v1"
)

var IngressNginxClassName = "nginx"
var IngressPathType = netv1.PathTypePrefix

func newDeployment(sd *deploymentv1.SingleDeployment) (*appsv1.Deployment, error) {
	deploy := newBaseDeployment(sd.Name, sd.Namespace)
	deploy.Spec.Replicas = &sd.Spec.Replicas
	deploy.Spec.Template.Spec.Containers = []corev1.Container{
		newBaseContainer(
			sd.Name,
			sd.Spec.Image,
			sd.Spec.Port),
	}

	return &deploy, nil
}

func newService(sd *deploymentv1.SingleDeployment) (*corev1.Service, error) {
	service := newBaseService(sd.Name, sd.Namespace)
	servicePort := newBaseServicePort("http", "TCP", sd.Spec.Expose.ServicePort, sd.Spec.Port)
	switch strings.ToLower(sd.Spec.Expose.Mode) {
	case ServiceNodePort:
		service.Spec.Type = corev1.ServiceTypeNodePort
		withNodePort(&servicePort, sd.Spec.Expose.NodePort)
		service.Spec.Ports = []corev1.ServicePort{servicePort}
	case ServiceIngress:
		service.Spec.Ports = []corev1.ServicePort{servicePort}
	default:
		return nil, field.Invalid(field.NewPath("spec").Child("expose", "mode"), sd.Spec.Expose.Mode, "not be support. Must is `NodePort` or `Ingress`")
	}

	return &service, nil
}

func newIngress(sd *deploymentv1.SingleDeployment) (*netv1.Ingress, error) {
	ingress := newBaseIngress(sd.Name, sd.Namespace)

	rule := newIngressBaseRule(sd.Spec.Expose.IngressDomain)
	httpPath := newIngressRuleHttpBasePath(sd.Name, sd.Spec.Expose.ServicePort)
	rule.HTTP.Paths = []netv1.HTTPIngressPath{httpPath}
	ingress.Spec.Rules = []netv1.IngressRule{rule}

	return &ingress, nil
}

func newBaseDeployment(name string, namespace string) appsv1.Deployment {
	d := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
	}
	d.ObjectMeta.Name = name
	d.ObjectMeta.Namespace = namespace

	nameMap := map[string]string{"app": name}
	d.ObjectMeta.Labels = nameMap
	d.Spec.Selector = &metav1.LabelSelector{}
	d.Spec.Selector.MatchLabels = nameMap
	d.Spec.Template.ObjectMeta.Labels = nameMap

	return d
}

func newBaseService(name string, namespace string) corev1.Service {
	s := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
	}
	s.ObjectMeta.Name = name
	s.ObjectMeta.Namespace = namespace

	s.Spec.Selector = map[string]string{"app": name}

	return s
}

func newBaseIngress(name string, namespace string) netv1.Ingress {
	i := netv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
	}
	i.ObjectMeta.Name = name
	i.ObjectMeta.Namespace = namespace
	i.Spec.IngressClassName = &IngressNginxClassName

	return i
}

func newBaseContainer(name, image string, port int32) corev1.Container {
	c := corev1.Container{}
	c.Name = name
	c.Image = image
	c.Ports = []corev1.ContainerPort{
		corev1.ContainerPort{
			ContainerPort: port,
		},
	}

	return c
}

func newBaseServicePort(name, protocol string, port, targetPort int32) corev1.ServicePort {
	sp := corev1.ServicePort{}
	sp.Name = name
	sp.Protocol = corev1.Protocol(protocol)
	sp.Port = port
	sp.TargetPort = intstr.IntOrString{
		Type:   intstr.Int,
		IntVal: targetPort,
	}

	return sp
}

func withNodePort(port *corev1.ServicePort, nodeport int32) {
	port.NodePort = nodeport
}

func newIngressBaseRule(domainHost string) netv1.IngressRule {
	r := netv1.IngressRule{}
	r.Host = domainHost
	r.HTTP = &netv1.HTTPIngressRuleValue{}
	return r
}

func newIngressRuleHttpBasePath(backendServiceName string, portNumber int32) netv1.HTTPIngressPath {
	p := netv1.HTTPIngressPath{}
	p.Path = "/"
	p.PathType = &IngressPathType
	p.Backend.Service = &netv1.IngressServiceBackend{
		Name: backendServiceName,
		Port: netv1.ServiceBackendPort{
			Number: portNumber,
		},
	}

	return p
}
