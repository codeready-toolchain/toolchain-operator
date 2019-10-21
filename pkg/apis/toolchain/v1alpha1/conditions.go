package v1alpha1

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

const (
	// status condition type
	CheReady       toolchainv1alpha1.ConditionType = "CheReady"
	CheNotReady    toolchainv1alpha1.ConditionType = "CheNotReady"
	TektonReady    toolchainv1alpha1.ConditionType = "TektonReady"
	TektonNotReady toolchainv1alpha1.ConditionType = "TektonNotReady"

	// Status condition reasons
	FailedToCreateCheSubscriptionReason    = "FailedToCreateCheSubscription"
	FailedToCreateTektonSubscriptionReason = "FailedToCreateTektonSubscription"
	CreatedCheSubscriptionReason           = "CreatedCheSubscription"
	CreatedTektonSubscriptionReason        = "CreatedTektonSubscription"
)

func SubscriptionCreated(conditionType toolchainv1alpha1.ConditionType, reason, message string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    conditionType,
		Status:  v1.ConditionTrue,
		Reason:  reason,
		Message: message,
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
