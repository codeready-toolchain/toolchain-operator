package v1alpha1

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

const (
	// status condition type
	CheReady    toolchainv1alpha1.ConditionType = "CheReady"
	TektonReady toolchainv1alpha1.ConditionType = "TektonReady"

	// Status condition reasons
	FailedToInstallReason = "FailedToInstall"
	InstalledReason       = "Installed"
)

func SubscriptionCreated(conditionType toolchainv1alpha1.ConditionType, reason string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:   conditionType,
		Status: v1.ConditionTrue,
		Reason: reason,
	}
}

func SubscriptionFailed(conditionType toolchainv1alpha1.ConditionType, reason, message string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    conditionType,
		Status:  v1.ConditionFalse,
		Reason:  reason,
		Message: message,
	}
}
