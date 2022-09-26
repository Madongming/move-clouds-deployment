package framework

import (
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func LoadYAML(file string, obj interface{}) (err error) {
	var data []byte
	if data, err = os.ReadFile(file); err != nil {
		return
	}
	err = yaml.Unmarshal(data, obj)
	return
}

func LoadYAMLToUnstructured(file string, obj *unstructured.Unstructured) (err error) {
	var data []byte
	if data, err = os.ReadFile(file); err != nil {
		return
	}
	err = yaml.Unmarshal(data, &(obj.Object))
	return
}
