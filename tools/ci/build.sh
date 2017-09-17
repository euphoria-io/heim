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
  if [ ${DRONE_BRANCH%/*} == 'dev' ]; then
    export HEIM_PREFIX="/$DRONE_BRANCH"
  fi
  pushd ./client
  gulp build
  popd

  go install -ldflags "-X main.version ${DRONE_COMMIT}" euphoria.io/heim/heimctl
  go install euphoria.io/heim/heimlich

  mv ./client/build/heim ${HEIM_GOPATH}/bin/static
  mv ./client/build/embed ${HEIM_GOPATH}/bin/embed
  mv ./client/build/email ${HEIM_GOPATH}/bin/email
  pushd ${HEIM_GOPATH}/bin
  generate_manifest static
  generate_manifest embed
  generate_manifest email
  find static embed email -type f | xargs heimlich heimctl

  s3cmd put heimctl.hzp s3://heim-release/${DRONE_COMMIT}
  if [ ${DRONE_BRANCH} == master ]; then
    s3cmd cp s3://heim-release/${DRONE_COMMIT} s3://heim-release/latest
  fi

  if [ ${DRONE_BRANCH%/*} == dev ]; then
    s3cmd cp s3://heim-release/${DRONE_COMMIT} s3://heim-release/${DRONE_BRANCH}
  fi

  popd
}

build_release
