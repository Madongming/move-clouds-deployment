package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	// ConditionStatusReady indicates the condition has been reached
	ConditionStatusReady = "True"
	// ConditionStatusReady indicates the result of condition is unknown
	ConditionStatusUnKnown = "UnKnown"
	// ConditionStatusFail indicates the condition has not been reached and it is a failure state
	ConditionStatusFailed = "False"
)

const (
	ConditionReason = ""
)

// Condition save the condition info for every condition when call deployment, statefulset and service
type Condition struct {
	// Type indicate which type this condition is. it can be deployment, statefulset or service
	Type string `json:"type"`

	// Message indicate the message of this condition. When status is false it must exist
	// +optional
	Message string `json:"message,omitempty"`

	// Status indicate the status of this condition. It can be true or false
	Status string `json:"status"`

	// Reason describe why this condition is not ready
	// +optional
	Reason string `json:"reason,omitempty"`

	// LastTransitionTime indicate the time when this condition happen to create or update
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// LastProbeTime indicate the time when check the status of this condition
	LastProbeTime metav1.Time `json:"lastProbeTime"`
}
