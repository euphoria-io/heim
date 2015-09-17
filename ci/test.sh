#!/bin/bash
set -ex

# pwd is gopath/src/euphoria-io/heim
export HEIM_GOPATH=$(pwd)/../../..

setup_deps() {
  git submodule update --init
  # required for running gulp out of this directory.
  ln -s $(pwd)/_deps/node_modules ./node_modules
  export PATH=${PATH}:$(pwd)/node_modules/.bin:$(pwd)/_deps/godeps/bin:${HEIM_GOPATH}/bin
  export GOPATH=${HEIM_GOPATH}:$(pwd)/_deps/godeps
}

test_backend() {
  psql -V
  psql -c 'create database heimtest;' -U postgres -h $DB_HOST
  export DSN="postgres://postgres@$DB_HOST/heimtest?sslmode=disable"
  go install github.com/coreos/etcd
  go test -v euphoria.io/heim/...
}

test_client() {
  export NODE_ENV=development
  pushd ./client
  gulp lint
  mochify
  popd
}

setup_deps

test_client
test_backend

if [ "$1" == "build" ]; then
  $(dirname $0)/build.sh
fi
