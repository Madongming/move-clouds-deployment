package controllers

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	deploymentv1 "github.com/Madongming/move-clouds-deployment/api/v1"
)

func getCondition(conds []deploymentv1.Condition, condType string) (*deploymentv1.Condition, int, bool) {
	for index := range conds {
		if conds[index].Type == condType {
			return &conds[index], index, true
		}
	}
	return nil, -1, false
}

func newCondition(conditionType, message, status, reason string) deploymentv1.Condition {
	return deploymentv1.Condition{
		Type:               conditionType,
		Message:            message,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}
}
