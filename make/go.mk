# By default the project should be build under GOPATH/src/github.com/<orgname>/<reponame>
GO_PACKAGE_ORG_NAME ?= $(shell basename $$(dirname $$PWD))
GO_PACKAGE_REPO_NAME ?= $(shell basename $$PWD)
GO_PACKAGE_PATH ?= github.com/${GO_PACKAGE_ORG_NAME}/${GO_PACKAGE_REPO_NAME}

GO111MODULE?=on
export GO111MODULE

.PHONY: build
## Build the operator
build: generate-assets $(OUT_DIR)/operator

$(OUT_DIR)/operator:
	$(Q)CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
		go build ${V_FLAG} \
		-ldflags "-X ${GO_PACKAGE_PATH}/cmd/manager.Commit=${GIT_COMMIT_ID} -X ${GO_PACKAGE_PATH}/cmd/manager.BuildTime=${BUILD_TIME}" \
		-o $(OUT_DIR)/bin/toolchain-operator \
		cmd/manager/main.go

.PHONY: vendor
vendor:
	$(Q)go mod vendor

TEKTON_INSTALLATION_CR_DIR=deploy/tekton

.PHONY: generate-assets
generate-assets:
	@echo "generating assets bindata..."
	@go install github.com/go-bindata/go-bindata/
	@$(GOPATH)/bin/go-bindata -pkg tekton -o pkg/resources/tekton/tekton_assets.go -nocompress -prefix $(TEKTON_INSTALLATION_CR_DIR) $(TEKTON_INSTALLATION_CR_DIR)
