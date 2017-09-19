#!/bin/bash
set -ex

# pwd is gopath/src/euphoria-io/heim
export HEIM_GOPATH=$(pwd)/../../..

setup_deps() {
  git submodule update --init
  export PATH=${PATH}:$(pwd)/node_modules/.bin:${HEIM_GOPATH}/bin
  export GOPATH=${HEIM_GOPATH}
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
  ETCD_VER=v3.2.7
  curl -L https://storage.googleapis.com/etcd/$ETCD_VER/etcd-$ETCD_VER-linux-amd64.tar.gz -o etcd.tar.gz
  mkdir -p etcd
  tar xzvf etcd.tar.gz -C etcd --strip-components=1
  sudo cp etcd/etcd etcd/etcdctl /usr/bin
}

setup_psql() {
  psql -V
  psql -c 'create database heimtest;' -U postgres -h $DB_HOST
  export DSN="postgres://postgres@$DB_HOST/heimtest?sslmode=disable"
}

setup_bazel
setup_deps
setup_etcd
setup_psql
bazel query 'kind(".*_test", ...) except //vendor/...' |
  xargs bazel test --test_output=errors --test_verbose_timeout_warnings --test_env=DSN="$DSN"

if [ "$1" == "build" ]; then
  $(dirname $0)/build.sh
fi
