#!/bin/bash

set -ex

go install heim/cmd/heim-backend

control_flags=

if [ -f /keys/devkey -a -f /keys/authorized_hosts ]; then
    control_flags="-control-hostkey /keys/devkey -control-authkeys /keys/authorized_hosts"
fi

# /go/src/heim/backend/static should be provided as a volume
# psql host should be provided as a linked container
/go/bin/heim-backend \
    -static /srv/heim/client/src/build \
    -http :80 \
    -psql 'postgres://postgres:heimlich@psql/heim?sslmode=disable' \
    $control_flags
