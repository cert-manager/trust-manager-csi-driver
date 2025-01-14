.PHONY: test-unit
## Unit tests
## @category Testing
test-unit: | $(NEEDS_GO) $(NEEDS_GOTESTSUM) $(ARTIFACTS)
	$(GOTESTSUM) \
		--junitfile=$(ARTIFACTS)/junit-go-e2e.xml \
		-- \
		-coverprofile=$(ARTIFACTS)/filtered.cov \
		./cmd/... ./internal/... \
		-- \
		-ldflags $(go_manager_ldflags)
	
	$(GO) tool cover -html=$(ARTIFACTS)/filtered.cov -o=$(ARTIFACTS)/filtered.html