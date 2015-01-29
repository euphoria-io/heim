#!/bin/bash

set -ex

go get heim/cmd/heim-backend
go install heim/cmd/heim-backend

# /go/src/heim/backend/static should be provided as a volume
# psql host should be provided as a linked container
/go/bin/heim-backend \
    -static /srv/heim/client/src/build \
    -http :80 \
    -psql 'postgres://postgres:heimlich@psql/heim?sslmode=disable'
