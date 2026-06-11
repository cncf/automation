TF ?= terraform
TF_ARGS ?=
TF_LOCK_TIMEOUT ?= 30s

# You have to provide the cluster name for some commands
OKE_CLUSTER ?= 

# You have to provide the bucket group name for some commands
BUCKETS ?= 

check_defined = $(if $(value OKE_CLUSTER),,$(error OKE_CLUSTER is not set))
check_buckets_defined = $(if $(value BUCKETS),,$(error BUCKETS is not set))

TF_VAR_tenancy_ocid ?=
TF_VAR_user_ocid ?=
TF_VAR_fingerprint ?=
TF_VAR_private_key_path ?=

check_compartment_ocid = $(if $(value TF_VAR_compartment_ocid),,$(error TF_VAR_compartment_ocid is not set))
check_tenancy_ocid = $(if $(value TF_VAR_tenancy_ocid),,$(error TF_VAR_tenancy_ocid is not set))
check_user_ocid = $(if $(value TF_VAR_user_ocid),,$(error TF_VAR_user_ocid is not set))
check_fingerprint = $(if $(value TF_VAR_fingerprint),,$(error TF_VAR_fingerprint is not set))
check_private_key_path = $(if $(value TF_VAR_private_key_path),,$(error TF_VAR_private_key_path is not set))

ensure-variables:
	@:$(call check_compartment_ocid, TF_VAR_compartment_ocid, Compartment OCID)
	@:$(call check_tenancy_ocid, TF_VAR_tenancy_ocid, Tenancy OCID)
	@:$(call check_user_ocid, TF_VAR_user_ocid, User OCID)
	@:$(call check_fingerprint, TF_VAR_fingerprint, Fingerprint)
	@:$(call check_private_key_path, TF_VAR_private_key_path, Private key path)

##@ Helpers:

.PHONY: help
help: ## Display this help.
	awk \
		-v "col=${COLOR}" -v "nocol=${NOCOLOR}" \
		' \
		BEGIN { \
			FS = ":.*##" ; \
			printf "\nUsage:\n  make %s<target>%s\n", col, nocol \
		} \
		/^[a-zA-Z_-]+:.*?##/ { \
			printf "  %s%-15s%s %s\n", col, $$1, nocol, $$2 \
		} \
		/^##@/ { \
			printf "\n%s%s%s\n", col, substr($$0, 5), nocol \
			} \
		' $(MAKEFILE_LIST)

### Clusters
cluster-init: ## Initialize Terraform's state and download necessary providers for OKE.
	@:$(call check_defined, OKE_CLUSTER, cluster name)
	@${MAKE} ensure-variables
	$(TF) -chdir=./cluster \
		init \
		-backend-config=./tfbackends/$(OKE_CLUSTER).tfbackend

cluster-plan: ## Present plan for creating/updating Terraform resources.
	@:$(call check_defined, OKE_CLUSTER, cluster name)
	@${MAKE} ensure-variables
	$(TF) -chdir=./cluster \
		plan $(TF_ARGS) \
		-out=plan.out \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		-var-file=./tfvars/$(OKE_CLUSTER).tfvars

cluster-apply: ## Create/Update Terraform resources.
	@${MAKE} ensure-variables
	$(TF) -chdir=./cluster \
		apply $(TF_ARGS) \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		"plan.out"
	rm -f ./cluster/plan.out

cluster-destroy: ## Delete Terraform resources.
	@:$(call check_defined, OKE_CLUSTER, cluster name)
	@${MAKE} ensure-variables
	$(TF) -chdir=./cluster \
		destroy $(TF_ARGS) \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		-var-file=./tfvars/$(OKE_CLUSTER).tfvars

.PHONY: output
cluster-output: ## Print Terraform output.
	@$(TF) -chdir=./cluster output -json

### Buckets
buckets-init: ## Initialize Terraform's state and download necessary providers for buckets.
	@:$(call check_buckets_defined, BUCKETS, bucket group name)
	@${MAKE} ensure-variables
	$(TF) -chdir=./buckets \
		init \
		-backend-config=./tfbackends/$(BUCKETS).tfbackend

buckets-plan: ## Present plan for creating/updating bucket resources.
	@:$(call check_buckets_defined, BUCKETS, bucket group name)
	@${MAKE} ensure-variables
	$(TF) -chdir=./buckets \
		plan $(TF_ARGS) \
		-out=plan.out \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		-var-file=./tfvars/$(BUCKETS).tfvars

buckets-apply: ## Create/Update bucket resources.
	@${MAKE} ensure-variables
	$(TF) -chdir=./buckets \
		apply $(TF_ARGS) \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		"plan.out"
	rm -f ./buckets/plan.out

buckets-destroy: ## Delete bucket resources.
	@:$(call check_buckets_defined, BUCKETS, bucket group name)
	@${MAKE} ensure-variables
	$(TF) -chdir=./buckets \
		destroy $(TF_ARGS) \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		-var-file=./tfvars/$(BUCKETS).tfvars

buckets-output: ## Print bucket Terraform output.
	@$(TF) -chdir=./buckets output -json

### Clean up
.PHONY: clean clean-cluster clean-buckets
clean: clean-cluster clean-buckets ## Clean up all Terraform cache and local state.

clean-cluster: ## Clean up cluster Terraform cache and local state.
	rm -rf ./cluster/.terraform ./cluster/.terraform.lock.hcl ./cluster/plan.out

clean-buckets: ## Clean up buckets Terraform cache and local state.
	rm -rf ./buckets/.terraform ./buckets/.terraform.lock.hcl ./buckets/plan.out
