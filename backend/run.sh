#!/bin/bash

set -ex

go install \
    -ldflags "-X main.version `git --git-dir=src/euphoria.io/heim/.git rev-parse HEAD`" \
    euphoria.io/heim/heimctl

control_flags=

if [ -f /keys/devkey -a -f /keys/authorized_hosts ]; then
    control_flags="-control-hostkey /keys/devkey -control-authkeys /keys/authorized_hosts"
fi

/go/bin/heimctl \
  -etcd-host http://etcd:4001 \
  -etcd /dev/euphoria.io \
  -config /go/src/euphoria.io/heim/heim.yml \
  serve -http :80 -console :2222 -static /srv/heim/client/src/build
