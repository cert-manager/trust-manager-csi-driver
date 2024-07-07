#!/bin/bash

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