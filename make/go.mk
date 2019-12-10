# By default the project should be build under GOPATH/src/github.com/<orgname>/<reponame>
GO_PACKAGE_ORG_NAME ?= $(shell basename $$(dirname $$PWD))
GO_PACKAGE_REPO_NAME ?= $(shell basename $$PWD)
GO_PACKAGE_PATH ?= github.com/${GO_PACKAGE_ORG_NAME}/${GO_PACKAGE_REPO_NAME}

GO111MODULE?=on
export GO111MODULE

.PHONY: build
## Build the operator
build: $(OUT_DIR)/operator generate-csv

$(OUT_DIR)/operator:
	$(Q)CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
		go build ${V_FLAG} \
		-ldflags "-X ${GO_PACKAGE_PATH}/cmd/manager.Commit=${GIT_COMMIT_ID} -X ${GO_PACKAGE_PATH}/cmd/manager.BuildTime=${BUILD_TIME}" \
		-o $(OUT_DIR)/bin/toolchain-operator \
		cmd/manager/main.go

.PHONY: vendor
vendor:
	$(Q)go mod vendor

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