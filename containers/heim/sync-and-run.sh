#!/bin/bash

set -ex

s3cmd -f get s3://heim-release/${REVISION} /backend.hzp
chmod +x /backend.hzp
/backend.hzp \
    -http :80 \
    -static /static \
    -psql 'postgres://postgres:heimlich@psql/heim?sslmode=disable'
