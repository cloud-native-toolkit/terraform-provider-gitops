#!/usr/bin/env bash

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

GIT_REPO=$(cat .git_repo)
GIT_TOKEN=$(cat .git_token)

BIN_DIR=$(cat .bin_dir)

DEST1=$(cat .dest1)
DEST2=$(cat .dest2)

export PATH="${BIN_DIR}:${PATH}"

source "${SCRIPT_DIR}/validation-functions.sh"

NAMESPACE=$(cat .namespace)

mkdir -p .testrepo

git clone "https://${GIT_TOKEN}@${GIT_REPO}" .testrepo

cd .testrepo || exit 1

find . -name "*"

set -e

ls "${DEST1}"
ls "${DEST1}" | while read file; do
  cat "${DEST1}/${file}"
done

ls "${DEST2}"
ls "${DEST2}" | while read file; do
  cat "${DEST2}/${file}"
done

#validate_gitops_ns_content "${NAMESPACE}" "${SERVER_NAME}"
#validate_gitops_ns_content "${NAMESPACE}" "${SERVER_NAME}" values.yaml
#validate_gitops_content "${NAMESPACE}" "test-rbac" "Chart.yaml" "1-infrastructure"
#validate_gitops_content "${NAMESPACE}" "test-rbac" "values.yaml" "1-infrastructure"
#validate_gitops_content "${NAMESPACE}" "sonarqube" "values.yaml" "2-services"
#validate_gitops_content "${NAMESPACE}" "dashboard" "values.yaml" "2-services"

cd ..
rm -rf .testrepo
