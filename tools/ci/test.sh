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

setup_bazel() {
  V=0.5.3
  OS=linux
  URL="https://github.com/bazelbuild/bazel/releases/download/${V}/bazel-${V}-installer-${OS}-x86_64.sh"
  wget -O install.sh "${URL}"
  chmod +x install.sh
  ./install.sh --user
  rm -f install.sh
}

setup_etcd() {
  curl -L https://storage.googleapis.com/etcd/$ETCD_VER/etcd-$ETCD_VER-linux-amd64.tar.gz -o etcd.tar.gz
  mkdir -p etcd
  tar xzvf /tmp/etcd.tar.gz -C /tmp/etcd --strip-components=1
  sudo cp /tmp/etcd/etcd /tmp/etcd/etcdctl /usr/bin
}

setup_psql() {
  psql -V
  psql -c 'create database heimtest;' -U postgres -h $DB_HOST
  export DSN="postgres://postgres@$DB_HOST/heimtest?sslmode=disable"
}

test_backend() {
  bazel query 'kind(".*_test", ...) except //vendor/...' |
    xargs bazel test --test_output=errors --test_verbose_timeout_warnings --test_env=DSN="$DSN"
}

test_client() {
  export NODE_ENV=development
  pushd ./client
  eslint ./
  mochify
  popd
}

setup_bazel
setup_deps
setup_etcd
setup_psql

test_client
test_backend

if [ "$1" == "build" ]; then
  $(dirname $0)/build.sh
fi
