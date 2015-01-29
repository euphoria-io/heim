#!/bin/bash

set -ex

go get heim/cmd/heim-upgradedb
go install heim/cmd/heim-upgradedb

/go/bin/heim-upgradedb \
    -psql 'postgres://postgres:heimlich@psql/heim?sslmode=disable'
