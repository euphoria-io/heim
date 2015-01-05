#!/bin/bash

set -ex

PATH=${PATH}:/var/cache/drone/bin
SRCDIR=/var/cache/drone/src

test_backend() {
  ln -sf /var/cache/drone/src/github.com/euphoria-io/heim ${SRCDIR}/heim
  go get -t heim/backend heim/backend/persist
  go test -v heim/backend

  psql -c 'create database heimtest;' -U postgres -h $POSTGRES_PORT_5432_TCP_ADDR
  go test -v heim/backend/persist --dsn "postgres://postgres@$POSTGRES_PORT_5432_TCP_ADDR/heimtest?sslmode=disable"
}

test_client() {
  export NODE_ENV=development
  cd ${SRCDIR}/heim/client
  npm install
  PATH=${PATH}:${SRCDIR}/heim/client/node_modules/.bin
  ln -s ${SRCDIR}/heim/client src
  cd src
  npm test
}

build_release() {
  export NODE_ENV=
  cd ${SRCDIR}/heim/client
  gulp build

  go get heim/backend/cmd/heimlich
  go install -ldflags "-X main.version ${DRONE_COMMIT}" heim/backend/cmd/heim-backend
  go install heim/backend/cmd/heimlich

  mv ${SRCDIR}/heim/client/build /var/cache/drone/bin/static
  cd /var/cache/drone/bin
  find static -type f | xargs heimlich heim-backend

  DEBIAN_FRONTEND=noninteractive apt-get install -y s3cmd
  cat > /root/.s3cfg << EOF
[default]
access_key = [redacted ;)]
secret_key = [redacted :O]
EOF
  s3cmd put heim-backend.hzp s3://heim-release/${DRONE_COMMIT}
  if [ ${DRONE_BRANCH} == master ]; then
    s3cmd cp s3://heim-release/${DRONE_COMMIT} s3://heim-release/latest
  fi

  if [ ${DRONE_BRANCH%/*} == logan ]; then
    s3cmd cp s3://heim-release/${DRONE_COMMIT} s3://heim-release/${DRONE_BRANCH}
  fi

  if [ ${DRONE_BRANCH%/*} == chromakode ]; then
    s3cmd cp s3://heim-release/${DRONE_COMMIT} s3://heim-release/${DRONE_BRANCH}
  fi
}

test_backend
test_client

build_release
