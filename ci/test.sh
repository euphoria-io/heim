#!/bin/bash
set -ex

# pwd is gopath/src/euphoria-io/heim
export HEIM_GOPATH=$(pwd)/../../..

setup_deps() {
  (cd client; npm install)
  export PATH=$(pwd)/client/node_modules/.bin:${HEIM_GOPATH}/bin:${PATH}
  export GOPATH=${HEIM_GOPATH}
  ls -alF $(pwd)/client/node_modules/.bin
}

test_backend() {
  psql -V
  psql -c 'create database heimtest;' -U postgres -h $DB_HOST
  export DSN="postgres://postgres@$DB_HOST/heimtest?sslmode=disable"
  go get github.com/coreos/etcd
  go install github.com/coreos/etcd
  go test -v euphoria.io/heim/...
}

test_client() {
  export NODE_ENV=development
  pushd ./client
  eslint ./
  mochify
  popd
}

setup_deps

test_client
test_backend

if [ "$1" == "build" ]; then
  $(dirname $0)/build.sh
fi
