TF ?= terraform
TF_ARGS ?=
TF_LOCK_TIMEOUT ?= 30s

# You have to provide the cluster name for some commands
LKE_CLUSTER ?= 

check_defined = $(if $(value LKE_CLUSTER),,$(error LKE_CLUSTER is not set))

TF_VAR_linode_api_token ?=
AWS_ACCESS_KEY_ID	?=
AWS_SECRET_ACCESS_KEY ?=

check_linode_api_token = $(if $(value TF_VAR_linode_api_token),,$(error TF_VAR_linode_api_token is not set))
check_aws_access_key_id = $(if $(value AWS_ACCESS_KEY_ID),,$(error AWS_ACCESS_KEY_ID is not set))
check_aws_secret_access_key = $(if $(value AWS_SECRET_ACCESS_KEY),,$(error AWS_SECRET_ACCESS_KEY is not set))

ensure-variables:
	@:$(call check_linode_api_token, TF_VAR_linode_api_token, Linode API token)
	@:$(call check_aws_access_key_id, AWS_ACCESS_KEY_ID, AWS access key ID)
	@:$(call check_aws_secret_access_key, AWS_SECRET_ACCESS_KEY, AWS secret access key)

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
	@${MAKE} ensure-variables
	$(TF) -chdir=./cluster \
		init \
		-backend-config=./tfbackends/$(LKE_CLUSTER).tfbackend

cluster-plan: ## Present plan for creating/updating Terraform resources.
	@:$(call check_defined, LKE_CLUSTER, cluster name)
	@${MAKE} ensure-variables
	$(TF) -chdir=./cluster \
		plan $(TF_ARGS) \
		-out=plan.out \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		-var-file=./tfvars/$(LKE_CLUSTER).tfvars

cluster-apply: ## Create/Update Terraform resources.
	@${MAKE} ensure-variables
	$(TF) -chdir=./cluster \
		apply $(TF_ARGS) \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		"plan.out"
	rm -f ./cluster/plan.out

cluster-destroy: ## Delete Terraform resources.
	@:$(call check_defined, LKE_CLUSTER, cluster name)
	@${MAKE} ensure-variables
	$(TF) -chdir=./cluster \
		destroy $(TF_ARGS) \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		-var-file=./tfvars/$(LKE_CLUSTER).tfvars

.PHONY: output
cluster-output: ## Print Terraform output.
	@$(TF) -chdir=./cluster output -json

### Clean up
.PHONY: clean clean-cluster
clean: clean-cluster ## Clean up all Terraform cache and local state.

clean-cluster: ## Clean up cluster Terraform cache and local state.
	rm -rf ./cluster/.terraform ./cluster/.terraform.lock.hcl ./cluster/plan.out

