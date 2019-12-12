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
	$(Q)go run $(shell pwd)/vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go \
		crd:trivialVersions=true \
		paths=./pkg/apis/... \
		output:dir=deploy/crds
	# Delete two first lines of the CRD ("\n----\n") to make a single manifest file out of the original multiple manifest file
	@find deploy/crds -name "toolchain.*.yaml" -exec sed -i '' -e '1,2d' '{}' \; 

PATH_TO_GENERATE_FILE=../api/scripts/olm-catalog-generate.sh

PHONY: generate-csv
generate-csv:
	$(eval GENERATE_PARAMS = -pr ../toolchain-operator -on codeready-toolchain-operator --allnamespaces true)
ifneq ("$(wildcard $(PATH_TO_GENERATE_FILE))","")
	@echo "generating CSV using script from local api repo..."
	$(PATH_TO_GENERATE_FILE) ${GENERATE_PARAMS}
else
	@echo "generating CSV using script from GH api repo (using latest version in master)..."
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/api/master/scripts/olm-catalog-generate.sh | bash -s -- ${GENERATE_PARAMS}
endif


CSV_VERSION_TO_GENERATE := 0.1.0

PHONY: generate-release-manifest
generate-release-manifest:
	$(eval GENERATE_PARAMS = -pr ../toolchain-operator -on codeready-toolchain-operator  --next-version ${CSV_VERSION_TO_GENERATE})
ifneq ("$(wildcard $(PATH_TO_GENERATE_FILE))","")
	@echo "generating release manifest in ./manifest/ directory using script from local api repo..."
	../api/scripts/create-release-bundle.sh ${GENERATE_PARAMS}
else
	@echo "generating release manifest in ./manifest/ directory using script from GH api repo (using latest version in master)..."
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/api/master/scripts/create-release-bundle.sh | bash -s -- ${GENERATE_PARAMS}
endif