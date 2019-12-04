package toolchain

// Labels return a map with a single label key/value to use
// when creating the installation resources (namespace, subscription, etc.)
func Labels() map[string]string {
	return map[string]string{"provider": "toolchain-operator"}
}
