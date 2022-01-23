#!/bin/bash

ROOT=$(cd $(dirname $0)/../../; pwd)

set -o errexit
set -o nounset
set -o pipefail

export CA_BUNDLE=$(kubectl config view --raw --flatten -o json | jq -r '.clusters[] | select(.name == "'kubernetes'") | .cluster."certificate-authority-data"')#$(kubectl config current-context)
# export CA_BUNDLE=$(kubectl config view --raw --flatten -o json | jq -r '.clusters[] | select(.name == "'$(kubectl config current-context)'") | .cluster."certificate-authority-data"')

if command -v envsubst >/dev/null 2>&1; then
    envsubst
else
    sed -e "s|\${CA_BUNDLE}|${CA_BUNDLE}|g"
fi


# cat ./validatingwebhook.yaml  | ./webhook-patch-ca-bundle.sh  > validatingwebhook-ca-bundle.yaml
# cat ./mutatingwebhook.yaml  | ./webhook-patch-ca-bundle.sh > mutatingwebhook-ca-bundle.yaml
# sed -i "s/=/= /g" ./validatingwebhook-ca-bundle.yaml
# sed -i "s/=/= /g" ./mutatingwebhook-ca-bundle.yaml