#!/bin/bash
set -ex

generate_manifest() {
  echo 'Generating manifest...'
  (
    cd "$1"
    find . -path ./MANIFEST.txt -prune -o -type f -exec md5sum {} \; \
      | sed -e 's@^\([0-9a-f]\+\) \+\./\(.*\)$@\2\t\1@g' | tee MANIFEST.txt
  )
}

build_release() {
  export NODE_ENV=production
  if [ ${BRANCH%/*} == 'dev' ]; then
    export HEIM_PREFIX="/$BRANCH"
  fi

  # TODO(logan): Manifests?
  bazel build \
      --action_env=NODE_ENV="$NODE_ENV" \
      --action_env=HEIM_PREFIX="$HEIM_PREFIX" \
      //:heimctl

  # TODO(logan): Upload build
}

check_env() {
  if [ -z "$BRANCH" ]; then
    echo "BRANCH not set"
    exit 1
  fi
  if [ -z "$COMMIT" ]; then
    echo "COMMIT not set"
    exit 1
  fi
}

check_env
build_release
