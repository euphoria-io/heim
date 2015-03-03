#!/bin/bash

set -ex

PATH=${PATH}:/var/cache/drone/bin
SRCDIR=/var/cache/drone/src
DEPSDIR=/var/cache/heim-deps

setup_deps() {
  ${DEPSDIR}/deps.sh link ${SRCDIR}/heim
  # required for running gulp out of that directory.
  ln -s ${DEPSDIR}/node_modules ${SRCDIR}/heim/node_modules
  PATH=${PATH}:${SRCDIR}/heim/node_modules/.bin
  GOPATH=${SRCDIR}/heim/deps/godeps:${GOPATH}
}

test_backend() {
  cd ${SRCDIR}
  psql -c 'create database heimtest;' -U postgres -h $POSTGRES_PORT_5432_TCP_ADDR
  export DSN="postgres://postgres@$POSTGRES_PORT_5432_TCP_ADDR/heimtest?sslmode=disable"
  go install github.com/coreos/etcd
  PATH="${PATH}":${SRCDIR}/heim/deps/godeps/bin go test -v heim/...
}

test_client() {
  export NODE_ENV=development
  cd ${SRCDIR}/heim/client
  gulp lint && mochify
}

build_release() {
  export NODE_ENV=production
  cd ${SRCDIR}/heim/client
  gulp build

  go install -ldflags "-X main.version ${DRONE_COMMIT}" heim/cmd/heim-backend
  go install heim/cmd/heimlich

  mv ${SRCDIR}/heim/client/build /var/cache/drone/bin/static
  cd /var/cache/drone/bin
  find static -type f | xargs heimlich heim-backend

  s3cmd put heim-backend.hzp s3://heim-release/${DRONE_COMMIT}
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

mv ${SRCDIR}/github.com/euphoria-io/heim ${SRCDIR}

setup_deps

test_backend
test_client

build_release
