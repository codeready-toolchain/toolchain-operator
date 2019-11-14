DOCKER_REPO?=quay.io/codeready-toolchain
IMAGE_NAME?=toolchain-operator

TIMESTAMP:=$(shell date +%s)
TAG?=$(GIT_COMMIT_ID_SHORT)-$(TIMESTAMP)

# to watch all namespaces, keep namespace empty
APP_NAMESPACE ?= $(LOCAL_TEST_NAMESPACE)
LOCAL_TEST_NAMESPACE ?= "toolchain-operator"

.PHONY: up-local
## Run Operator locally
up-local: login-as-admin create-namespace deploy-rbac build deploy-crd
	$(Q)-oc new-project $(LOCAL_TEST_NAMESPACE) || true
	$(Q)operator-sdk up local --namespace=$(APP_NAMESPACE) --verbose

.PHONY: login-as-admin
## Log in as system:admin
login-as-admin:
    ifneq ($(IS_OS_3),)
		$(info logging as system:admin)
		oc login -u system:admin 1>/dev/null
    else ifneq ($(IS_CRC),)
        ifneq ($(IS_KUBE_ADMIN),)
			$(info logging as kube:admin)
			oc login -u=kubeadmin -p=`cat ~/.crc/cache/crc_libvirt_*/kubeadmin-password` 1>/dev/null
        endif
    endif

.PHONY: create-namespace
## Create the test namespace
create-namespace:
	$(Q)-echo "Creating Namespace"
	$(Q)-oc new-project $(LOCAL_TEST_NAMESPACE)

.PHONY: use-namespace
## Log in as system:admin and enter the test namespace
use-namespace: login-as-admin
	$(Q)-echo "Using to the namespace $(LOCAL_TEST_NAMESPACE)"
	$(Q)-oc project $(LOCAL_TEST_NAMESPACE)

.PHONY: clean-namespace
## Delete the test namespace
clean-namespace:
	$(Q)-echo "Deleting Namespace"
	$(Q)-oc delete project $(LOCAL_TEST_NAMESPACE)

.PHONY: reset-namespace
## Delete an create the test namespace and deploy rbac there
reset-namespace: login-as-admin clean-namespace create-namespace deploy-rbac

.PHONY: deploy-rbac
## Setup service account and deploy RBAC
deploy-rbac:
	$(Q)-oc apply -f deploy/service_account.yaml
	$(Q)-oc apply -f deploy/role.yaml
	$(Q)-oc apply -f deploy/role_binding.yaml
# 	$(Q)-oc apply -f deploy/cluster_role.yaml
# 	$(Q)-sed -e 's|REPLACE_NAMESPACE|${LOCAL_TEST_NAMESPACE}|g' ./deploy/cluster_role_binding.yaml  | oc apply -f -

.PHONY: deploy-crd
## Deploy CRD
deploy-crd:
	$(Q)-oc apply -f deploy/crds
