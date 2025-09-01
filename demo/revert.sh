#!/bin/bash

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

set -eu

GREEN='\033[1;32m'
NC='\033[0m' # No Color
function info { printf "${GREEN}[INFO] $@${NC}\n"; }

info "reverting bundle"

cat <<EOF | kubectl apply -f - -o yaml | pygmentize -l yaml | sed 's/^/~>  /'
apiVersion: trust.cert-manager.io/v1alpha1
kind: Bundle
metadata:
  name: example.com
spec:
  sources:
  - useDefaultCAs: true
  target:
    configMap:
      key: "root-certs.pem"
EOF