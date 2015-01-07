#!/bin/bash

set -ex

if [ ! -e ${HZPDIR}/backend.hzp ]; then
    rm -rf ${HZPDIR}
    mkdir -p ${HZPDIR}
    s3cmd -f get s3://heim-release/${REVISION} ${HZPDIR}/backend.hzp
    chmod +x ${HZPDIR}/backend.hzp
fi

if [ ! -e /host.pem ]; then
    ssh-keygen -t rsa -b 2048 -f /host.pem -q -N ""
fi

cd ${HZPDIR}
echo SSH key:
ssh-keygen -l -f /host.pem
./backend.hzp \
    -http :80 \
    -control :22 \
    -control-authkeys /authorized_keys \
    -control-hostkey /host.pem \
    -static /static \
    -psql 'postgres://postgres:heimlich@psql/heim?sslmode=disable'
