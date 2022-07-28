package framework

import (
	"io/ioutil"

	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"
)

// LoadFile loads a file and asserts its success. Useful to
// load test data or other necessary data from the OS without having to validate
func LoadFile(file string) (data []byte) {
	var err error
	data, err = ioutil.ReadFile(file)
	Expect(err).To(Succeed(), "should have loaded file "+file)
	return
}

// LoadYAML loads yaml
func LoadYAML(file string, obj interface{}) (err error) {
	var data []byte
	if data, err = ioutil.ReadFile(file); err != nil {
		return
	}
	err = yaml.Unmarshal(data, obj)
	return
}
