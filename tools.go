// +build tools

package tools

import (
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
	_ "k8s.io/code-generator/cmd/deepcopy-gen"

)
