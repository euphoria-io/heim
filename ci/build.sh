#!/bin/bash

set -ex

PATH=${PATH}:/var/cache/drone/bin
SRCDIR=/var/cache/drone/src
HEIMDIR=${SRCDIR}/euphoria.io/heim
DEPSDIR=/var/cache/heim-deps

setup_deps() {
  ${DEPSDIR}/deps.sh link ${HEIMDIR}
  # required for running gulp out of that directory.
  ln -s ${DEPSDIR}/node_modules ${HEIMDIR}/node_modules
  PATH=${PATH}:${HEIMDIR}/node_modules/.bin
  GOPATH=${HEIMDIR}/deps/godeps:${GOPATH}
}

test_backend() {
  psql -c 'create database heimtest;' -U postgres -h $POSTGRES_PORT_5432_TCP_ADDR
  export DSN="postgres://postgres@$POSTGRES_PORT_5432_TCP_ADDR/heimtest?sslmode=disable"
  go install github.com/coreos/etcd
  PATH="${PATH}":${HEIMDIR}/deps/godeps/bin go test -v euphoria.io/heim/...
}

test_client() {
  export NODE_ENV=development
  cd ${HEIMDIR}/client
  gulp lint && mochify
}

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
  cd ${HEIMDIR}/client
  gulp build

  go install -ldflags "-X main.version ${DRONE_COMMIT}" euphoria.io/heim/heimctl
  go install euphoria.io/heim/heimlich

  mv ${HEIMDIR}/client/build /var/cache/drone/bin/static
  cd /var/cache/drone/bin
  generate_manifest static
  find static -type f | xargs heimlich heimctl

  s3cmd put heimctl.hzp s3://heim-release/${DRONE_COMMIT}
  if [ ${DRONE_BRANCH} == master ]; then
    s3cmd cp s3://heim-release/${DRONE_COMMIT} s3://heim-release/latest
  fi

  if [ ${DRONE_BRANCH%/*} == logan ]; then
    s3cmd cp s3://heim-release/${DRONE_COMMIT} s3://heim-release/${DRONE_BRANCH}
  fi

  if [ ${DRONE_BRANCH%/*} == chromakode ]; then
    s3cmd cp s3://heim-release/${DRONE_COMMIT} s3://heim-release/${DRONE_BRANCH}
  fi
}

mv ${SRCDIR}/github.com/euphoria-io ${SRCDIR}/euphoria.io

setup_deps

test_backend
test_client

build_release
