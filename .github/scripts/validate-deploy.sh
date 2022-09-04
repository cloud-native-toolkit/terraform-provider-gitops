#!/usr/bin/env bash

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

GIT_REPO=$(cat .git_repo)
GIT_TOKEN=$(cat .git_token)

BIN_DIR=$(cat .bin_dir)

export PATH="${BIN_DIR}:${PATH}"

source "${SCRIPT_DIR}/validation-functions.sh"

NAMESPACE=$(cat .namespace)

mkdir -p .testrepo

git clone "https://${GIT_TOKEN}@${GIT_REPO}" .testrepo

cd .testrepo || exit 1

find . -name "*"

set -e

validate_gitops_ns_content "${NAMESPACE}" "${SERVER_NAME}"

cd ..
rm -rf .testrepo
