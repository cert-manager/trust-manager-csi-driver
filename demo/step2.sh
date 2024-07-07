#!/bin/bash

set -eu

GREEN='\033[1;32m'
NC='\033[0m' # No Color
function info { printf "${GREEN}[INFO] $@${NC}\n"; }

info "injecting ca into bundle"

cat <<EOF | kubectl apply -f - -o yaml | pygmentize -l yaml | sed 's/^/~>  /'
apiVersion: trust.cert-manager.io/v1alpha1
kind: Bundle
metadata:
  name: example.com
spec:
  sources:
  - useDefaultCAs: true
  - inLine: |
$(kubectl get secret -n example ca-secret -o yaml | yq '.data["tls.crt"] | @base64d' | awk '{ print "      " $0 }')
  target:
    configMap:
      key: "root-certs.pem"
EOF