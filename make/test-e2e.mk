# Copyright 2024 The cert-manager Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: e2e-setup-trust-manager
e2e-setup-trust-manager: e2e-setup-cert-manager | kind-cluster $(NEEDS_HELM)
	$(HELM) upgrade \
		--install \
		--create-namespace \
		--wait \
		--version $(quay.io/jetstack/trust-manager.TAG) \
		--namespace cert-manager \
		--repo https://charts.jetstack.io \
		--set image.repository=$(quay.io/jetstack/trust-manager.REPO) \
		--set image.tag=$(quay.io/jetstack/trust-manager.TAG) \
		--set image.pullPolicy=Never \
		trust-manager trust-manager >/dev/null

.PHONY: e2e-setup-cert-manager
e2e-setup-cert-manager: | kind-cluster $(NEEDS_HELM) $(NEEDS_KUBECTL)
	$(HELM) upgrade \
		--install \
		--create-namespace \
		--wait \
		--version $(quay.io/jetstack/cert-manager-controller.TAG) \
		--namespace cert-manager \
		--repo https://charts.jetstack.io \
		--set installCRDs=true \
		--set image.repository=$(quay.io/jetstack/cert-manager-controller.REPO) \
		--set image.tag=$(quay.io/jetstack/cert-manager-controller.TAG) \
		--set image.pullPolicy=Never \
		--set cainjector.image.repository=$(quay.io/jetstack/cert-manager-cainjector.REPO) \
		--set cainjector.image.tag=$(quay.io/jetstack/cert-manager-cainjector.TAG) \
		--set cainjector.image.pullPolicy=Never \
		--set webhook.image.repository=$(quay.io/jetstack/cert-manager-webhook.REPO) \
		--set webhook.image.tag=$(quay.io/jetstack/cert-manager-webhook.TAG) \
		--set webhook.image.pullPolicy=Never \
		--set startupapicheck.image.repository=$(quay.io/jetstack/cert-manager-startupapicheck.REPO) \
		--set startupapicheck.image.tag=$(quay.io/jetstack/cert-manager-startupapicheck.TAG) \
		--set startupapicheck.image.pullPolicy=Never \
		cert-manager cert-manager >/dev/null

# The "install" target can be run on its own with any currently active cluster,
# we can't use any other cluster then a target containing "test-e2e" is run.
# When a "test-e2e" target is run, the currently active cluster must be the kind
# cluster created by the "kind-cluster" target.
ifeq ($(findstring test-e2e,$(MAKECMDGOALS)),test-e2e)
install: kind-cluster oci-load-manager
endif

test-e2e-deps: INSTALL_OPTIONS :=
test-e2e-deps: INSTALL_OPTIONS += --set image.repository=$(oci_manager_image_name_development) --set app.logLevel=5
test-e2e-deps: e2e-setup-trust-manager e2e-setup-cert-manager
test-e2e-deps: install

