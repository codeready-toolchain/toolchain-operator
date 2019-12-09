# current groupname and version of the operators'API
API_GROUPNAME=toolchain
API_VERSION:=v1alpha1
API_FULL_GROUPNAME=toolchain.openshift.dev

.PHONY: generate
## Generate deepcopy, openapi and CRD files after the API was modified
generate: vendor generate-deepcopy generate-openapi generate-crds generate-csv
	
.PHONY: generate-deepcopy
generate-deepcopy:
	@echo "re-generating the deepcopy go file..."
	$(Q)go run $(shell pwd)/vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go \
	--input-dirs ./pkg/apis/$(API_GROUPNAME)/$(API_VERSION)/ -O zz_generated.deepcopy \
	--bounding-dirs github.com/codeready-toolchain/toolchain-operator/pkg/apis "$(API_GROUPNAME):$(API_VERSION)" \
	--go-header-file=make/go-header.txt
	
.PHONY: generate-openapi
generate-openapi:
	@echo "re-generating the openapi go file..."
	$(Q)go run $(shell pwd)/vendor/k8s.io/kube-openapi/cmd/openapi-gen/openapi-gen.go \
	--input-dirs ./pkg/apis/$(API_GROUPNAME)/$(API_VERSION)/ \
	--output-package github.com/codeready-toolchain/toolchain-operator/pkg/apis/$(API_GROUPNAME)/$(API_VERSION) \
	--output-file-base zz_generated.openapi \
	--go-header-file=make/go-header.txt

.PHONY: generate-crds
generate-crds: vendor 
	@echo "Re-generating the Toolchain CRD files..."
	$(Q)go run $(shell pwd)/vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go crd:trivialVersions=true \
	paths=./pkg/apis/... output:dir=deploy/crds

PATH_TO_GENERATE_FILE=../api/scripts/olm-catalog-generate.sh

.PHONY: generate-csv
generate-csv:
ifneq ("$(wildcard $(PATH_TO_GENERATE_FILE))","")
	@echo "generating CSV using script from local api repo..."
	$(PATH_TO_GENERATE_FILE) -pr ../toolchain-operator/ -on codeready-toolchain-operator
else
	@echo "generating CSV using script from GH api repo (using latest version in master)..."
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/api/master/scripts/olm-catalog-generate.sh | bash -s --  -pr ../toolchain-operator/ -on codeready-toolchain-operator
endif
