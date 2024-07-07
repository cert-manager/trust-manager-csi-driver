include make/test-e2e.mk
include make/test-unit.mk

$(kind_cluster_config): make/config/kind/cluster.yaml | $(bin_dir)/scratch
	cat $< | \
	sed -e 's|{{KIND_IMAGES}}|$(CURDIR)/$(images_tar_dir)|g' \
	> $@

.PHONY: generate-conversion
generate-conversion: $(NEEDS_CONVERSION-GEN)
	$(CONVERSION-GEN)  \
      --go-header-file $(go_header_file) \
      --output-file zz_generated.conversion.go \
      $(addprefix $(repo_name)/,$(conversion_packages))

shared_generate_targets += generate-conversion