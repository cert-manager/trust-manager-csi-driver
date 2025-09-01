repo_name := github.com/cert-manager/trust-manager-csi-driver

# Trust manager version used for testing
trust_manager_version := v0.11.0
images_amd64 += quay.io/jetstack/ctrust-manager:$(trust_manager_version)@sha256:741aeece047c20f5a3eb128491a502eb9dc422da2b9a866784cde161919cb17b
images_arm64 += quay.io/jetstack/trust-manager:$(trust_manager_version)@sha256:ec2354a0a091896cbb60fa0219c5216dcd40cb97a841fe1e9a7495ec5c172a94

# Kind config
kind_cluster_name := trust-manager-csi-driver
kind_cluster_config := $(bin_dir)/scratch/kind_cluster.yaml

# Build config (global)
build_names := manager

# Build config (go)
go_manager_main_dir := ./cmd/csi-driver
go_manager_mod_dir := .
go_manager_ldflags := -X $(repo_name)/internal/version.AppVersion=$(VERSION) -X $(repo_name)/internal/version.GitCommit=$(GITCOMMIT)

# Build config (oci)
oci_manager_base_image_flavor := csi-static
oci_manager_image_name := quay.io/jetstack/trust-manager-csi-driver
oci_manager_image_tag := $(VERSION)
oci_manager_image_name_development := cert-manager.local/trust-manager-csi-driver

# Deploy config
deploy_name := trust-manager-csi-driver
deploy_namespace := cert-manager

# Code generation
conversion_packages = internal/api/metadata/v1alpha1

# api_docs_outfile := docs/api/api.md
# api_docs_package := $(repo_name)/pkg/apis/trust/v1alpha1
# api_docs_branch := main

# Lint
golangci_lint_config := .golangci.yaml

# Helm
helm_chart_source_dir := deploy/charts/trust-manager-csi-driver
helm_chart_name := trust-manager-csi-driver
helm_chart_image_name := quay.io/jetstack/charts/trust-manager-csi-driver
helm_chart_version := $(VERSION)
helm_labels_template_name := trust-manager-csi-driver.labels
helm_docs_use_helm_tool := 1
helm_generate_schema := 1
helm_verify_values := 1

define helm_values_mutation_function
$(YQ) \
	'( .image.repository = "$(oci_manager_image_name)" ) | \
	( .image.tag = "$(oci_manager_image_tag)" )' \
	$1 --inplace
endef