#!/bin/bash

set -ex

export NODE_ENV=development

SRCDIR=/var/cache/drone/src

test_backend() {
  ln -sf /var/cache/drone/src/github.com/euphoria-io/heim ${SRCDIR}/heim
  go get -t heim/backend heim/backend/persist
  go test -v heim/backend

  psql -c 'create database heimtest;' -U postgres -h $POSTGRES_PORT_5432_TCP_ADDR
  go test -v heim/backend/persist --dsn "postgres://postgres@$POSTGRES_PORT_5432_TCP_ADDR/heimtest?sslmode=disable"
}

test_client() {
  cd ${SRCDIR}/heim/client
  npm install
  PATH=${PATH}:${SRCDIR}/heim/client/node_modules/.bin
  ln -s ${SRCDIR}/heim/client src
  cd src
  npm test
}

test_backend
test_client
