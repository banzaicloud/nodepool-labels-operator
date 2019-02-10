set -o errexit
set -o nounset
set -o pipefail

if [[ -z "${GOPATH}" ]]; then
  echo "GOPATH must be set"
  exit 1
fi

if [[ ! -d "${GOPATH}/src/k8s.io/code-generator" ]]; then
  echo "k8s.io/code-generator missing from GOPATH"
  exit 1
fi

PROJECT_DIR=$(realpath $(dirname "${BASH_SOURCE}"))

cd ${GOPATH}/src/k8s.io/code-generator

./generate-groups.sh \
  all \
  github.com/banzaicloud/nodepool-labels-operator/pkg/client \
  github.com/banzaicloud/nodepool-labels-operator/pkg/apis \
  nodepoollabelset:v1alpha1 \
  --go-header-file ${PROJECT_DIR}/header.txt
