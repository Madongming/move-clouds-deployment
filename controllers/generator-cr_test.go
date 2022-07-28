package controllers

import (
	"embed"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	deploymentv1 "github.com/Madongming/move-clouds-deployment/api/v1"
	"github.com/Madongming/move-clouds-deployment/controllers/testdata"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func readFile(filename string) ([]byte, error) {
	file, err := os.Open(filepath.Join("testdata", filename))
	if err != nil {

	}
	defer file.Close()

	return ioutil.ReadAll(file)
}

func makeSingleDeployment(filename string) *deploymentv1.SingleDeployment {
	content, err := readFile(filename)
	if err != nil {
		panic(err)
	}

	sd := new(deploymentv1.SingleDeployment)
	if err := yaml.Unmarshal(content, sd); err != nil {
		panic(err)
	}

	return sd
}

func makeDeployment(filename string) *appsv1.Deployment {
	content, err := readFile(filename)
	if err != nil {
		panic(err)
	}

	d := new(appsv1.Deployment)
	if err := yaml.Unmarshal(content, d); err != nil {
		panic(err)
	}

	return d
}

func makeService(filename string) *corev1.Service {
	content, err := readFile(filename)
	if err != nil {
		panic(err)
	}

	svc := new(corev1.Service)
	if err := yaml.Unmarshal(content, svc); err != nil {
		panic(err)
	}

	return svc
}

func makeIngress(filename string) *netv1.Ingress {
	content, err := readFile(filename)
	if err != nil {
		panic(err)
	}

	ig := new(netv1.Ingress)
	if err := yaml.Unmarshal(content, ig); err != nil {
		panic(err)
	}

	return ig
}

func Test_newDeployment(t *testing.T) {
	type args struct {
		emfs embed.FS
		sd   *deploymentv1.SingleDeployment
	}
	tests := []struct {
		name    string
		args    args
		want    *appsv1.Deployment
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "Test case create ingress mode for deployment",
			args: args{
				emfs: testdata.EmTestdata,
				sd:   makeSingleDeployment("deployment_v1_singledeployment_rc_ingress.yaml"),
			},
			want:    makeDeployment("deployment_except_ingress.yaml"),
			wantErr: false,
		},
		{
			name: "Test case create nodeport mode for deployment",
			args: args{
				emfs: testdata.EmTestdata,
				sd:   makeSingleDeployment("deployment_v1_singledeployment_rc_nodeport.yaml"),
			},
			want:    makeDeployment("deployment_except_nodeport.yaml"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newDeployment(tt.args.emfs, tt.args.sd)
			if (err != nil) != tt.wantErr {
				t.Errorf("newDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newDeployment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newService(t *testing.T) {
	type args struct {
		emfs embed.FS
		sd   *deploymentv1.SingleDeployment
	}
	tests := []struct {
		name    string
		args    args
		want    *corev1.Service
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "Test case create ingress mode for service",
			args: args{
				emfs: testdata.EmTestdata,
				sd:   makeSingleDeployment("deployment_v1_singledeployment_rc_ingress.yaml"),
			},
			want:    makeService("service_except_ingress.yaml"),
			wantErr: false,
		},
		{
			name: "Test case create nodeport mode for service",
			args: args{
				emfs: testdata.EmTestdata,
				sd:   makeSingleDeployment("deployment_v1_singledeployment_rc_nodeport.yaml"),
			},
			want:    makeService("service_except_nodeport.yaml"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newService(tt.args.emfs, tt.args.sd)
			if (err != nil) != tt.wantErr {
				t.Errorf("newService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newIngress(t *testing.T) {
	type args struct {
		emfs embed.FS
		sd   *deploymentv1.SingleDeployment
	}
	tests := []struct {
		name    string
		args    args
		want    *netv1.Ingress
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "Test case create ingress mode for ingress",
			args: args{
				emfs: testdata.EmTestdata,
				sd:   makeSingleDeployment("deployment_v1_singledeployment_rc_ingress.yaml"),
			},
			want:    makeIngress("ingress_except_ingress.yaml"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newIngress(tt.args.emfs, tt.args.sd)
			if (err != nil) != tt.wantErr {
				t.Errorf("newIngress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newIngress() = %v, want %v", got, tt.want)
			}
		})
	}
}
