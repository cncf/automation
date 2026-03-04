# terraform 1.12 support oci backend, so it's used at the moment.
# we can switch back to opentofu when oci backend is supported
OPENTOFU_CLI ?= terraform

TF_VAR_tenant_ocid ?=
TF_VAR_user_ocid ?=
TF_VAR_fingerprint ?=
TF_VAR_private_key_path ?=

check_compartment_ocid = $(if $(value TF_VAR_compartment_ocid),,$(error TF_VAR_compartment_ocid is not set))
check_tenant_ocid = $(if $(value TF_VAR_tenant_ocid),,$(error TF_VAR_tenant_ocid is not set))
check_user_ocid = $(if $(value TF_VAR_user_ocid),,$(error TF_VAR_user_ocid is not set))
check_fingerprint = $(if $(value TF_VAR_fingerprint),,$(error TF_VAR_fingerprint is not set))
check_private_key_path = $(if $(value TF_VAR_private_key_path),,$(error TF_VAR_private_key_path is not set))

ensure-variables:
	@:$(call check_compartment_ocid, TF_VAR_compartment_ocid, Compartment OICD)
	@:$(call check_tenant_ocid, TF_VAR_tenant_ocid, Tenant OICD)
	@:$(call check_user_ocid, TF_VAR_user_ocid, User OICD)
	@:$(call check_fingerprint, TF_VAR_fingerprint, Fingerprint)
	@:$(call check_private_key_path, TF_VAR_private_key_path, Private key path)

init:
	@make ensure-variables
	$(OPENTOFU_CLI) init

fmt:
	$(OPENTOFU_CLI) fmt

plan:
	@make ensure-variables
	$(OPENTOFU_CLI) plan -out=plan.out

apply:
	@make ensure-variables
	$(OPENTOFU_CLI) apply "plan.out"
	rm -f plan.out

kubeconfig:
	$(OPENTOFU_CLI) output -json | jq -r '.cluster.value.kubeconfig' > kubeconfig
