/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SingleDeploymentSpec defines the desired state of SingleDeployment
type SingleDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of SingleDeployment. Edit singledeployment_types.go to remove/update
	// Foo string `json:"foo,omitempty"`

	// Port The port this instance accesses, and the port you want to expose
	Port int `json:"port"`

	// IngressDomain If there is a value, it means that the domain name is used to access through ingress, and the instance will be added to ingress and accessed through the unified portal.
	//+optional
	IngressDomain string `json:"ingressDomain,omitempty"`

	// Image The image used for deployment. If this item is empty, build will be used to build the image, so only one of this item and build can be empty. If both exist, this item will work
	//+optional
	Image string `json:"image,omitempty"`

	// Replicas How many replicas you want deployment, default is 1
	//+optional
	Replicas int32 `json:"replicas,omitempty"`

	// StartCmd Start command, if empty, use the buit-in CMD/ENTRYPOINT
	//+optional
	StartCmd string `json:"startCmd,omitempty"`

	// Args Parameter list for the startup command, if empty, use the buit-in CMD/ENTRYPOINT
	//+optional
	Args []string `json:"args,omitempty"`
}

// SingleDeploymentStatus defines the observed state of SingleDeployment
type SingleDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Phase Execution phase: Creating | Running | Success | Failed | Deleting
	// +optional
	Phase string `json:"phase,omitempty"`

	// Message Execution message
	// +optional
	Message string `json:"message,omitempty"`

	// Reason If it fails, what is the reason
	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Conditions []Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
/*
   //+kubebuilder:printcolumn:name="SomeRef",type=string,JSONPath=".spec.someRef.name"
*/
//+kubebuilder:resource:scope=Namespaced,shortName={dsg}
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SingleDeployment is the Schema for the singledeployments API
type SingleDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SingleDeploymentSpec   `json:"spec,omitempty"`
	Status SingleDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SingleDeploymentList contains a list of SingleDeployment
type SingleDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SingleDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SingleDeployment{}, &SingleDeploymentList{})
}
