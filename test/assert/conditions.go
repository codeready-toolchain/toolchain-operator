package assert

import (
	"fmt"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODo - This is copied as it is, due to issues of controller-runtime's client.Client. There is update on some methods of client.Client, which causes compile error. This can be removed once we updated controller-runtime for common
// https://github.com/operator-framework/operator-sdk/blob/master/doc/migration/version-upgrade-guide.md#controller-runtime-api-updates

// AssertConditionsMatch asserts that the specified list A of conditions is equal to specified
// list B of conditions ignoring the order of the elements. We can't use assert.ElementsMatch
// because the LastTransitionTime of the actual conditions can be modified but the conditions
// still should be treated as matched
func AssertConditionsMatch(t *testing.T, actual []toolchainv1alpha1.Condition, expected ...toolchainv1alpha1.Condition) {
	require.Equal(t, len(expected), len(actual))
	for _, c := range expected {
		AssertContainsCondition(t, actual, c)
	}
}

// AssertConditionsEmpty verifies that the actual conditions are empty
func AssertConditionsEmpty(t *testing.T, actual []toolchainv1alpha1.Condition) {
	require.Empty(t, actual)
}

// AssertContainsCondition asserts that the specified list of conditions contains the specified condition.
// LastTransitionTime is ignored.
func AssertContainsCondition(t *testing.T, conditions []toolchainv1alpha1.Condition, contains toolchainv1alpha1.Condition) {
	for _, c := range conditions {
		if c.Type == contains.Type {
			assert.Equal(t, contains.Status, c.Status)
			assert.Equal(t, contains.Reason, c.Reason)
			assert.Equal(t, contains.Message, c.Message)
			return
		}
	}
	assert.FailNow(t, fmt.Sprintf("the list of conditions %v doesn't contain the expected condition %v", conditions, contains))
}

// ConditionsMatch returns true if the specified list A of conditions is equal to specified
// list B of conditions ignoring the order of the elements
func ConditionsMatch(actual []toolchainv1alpha1.Condition, expected ...toolchainv1alpha1.Condition) bool {
	if len(expected) != len(actual) {
		return false
	}
	for _, c := range expected {
		if !ContainsCondition(actual, c) {
			return false
		}
	}
	for _, c := range actual {
		if !ContainsCondition(expected, c) {
			return false
		}
	}
	return true
}

// ContainsCondition returns true if the specified list of conditions contains the specified condition.
// LastTransitionTime is ignored.
func ContainsCondition(conditions []toolchainv1alpha1.Condition, contains toolchainv1alpha1.Condition) bool {
	for _, c := range conditions {
		if c.Type == contains.Type {
			return contains.Status == c.Status && contains.Reason == c.Reason && contains.Message == c.Message
		}
	}
	return false
}
