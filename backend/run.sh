#!/bin/bash

set -ex

go install \
    -ldflags "-X main.version `git --git-dir=src/euphoria.io/heim/.git rev-parse HEAD`" \
    euphoria.io/heim/cmd/heim-backend

control_flags=

if [ -f /keys/devkey -a -f /keys/authorized_hosts ]; then
    control_flags="-control-hostkey /keys/devkey -control-authkeys /keys/authorized_hosts"
fi

/go/bin/heim-backend \
  -http :80 \
  -console :2222 \
  -static /srv/heim/client/src/build \
  -etcd-host http://etcd:4001 \
  -etcd /dev/euphoria.io \
  -config /go/src/euphoria.io/heim/heim.yml

cat > /dev/null << EOM
# /srv/heim/client/src/build should be provided as a volume
# psql host should be provided as a linked container
/go/bin/heim-backend \
    -static /srv/heim/client/src/build \
    -http :80 \
    -psql 'postgres://postgres:heimlich@psql/heim?sslmode=disable' \
    -kms-local-key-file /keys/masterkey \
    -etcd-peers http://etcd:4001 -etcd /dev/euphoria.io \
    $control_flags
EOM
