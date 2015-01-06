#!/bin/bash

set -ex

cp -r /ssh /root/.ssh
chown -R root.root /root/.ssh
git clone git@github.com:euphoria-io/heim.git /go/src/heim
cd /go/src/heim
if [ ${REVISION} != latest ]; then
    git checkout ${REVISION}
fi

go get heim/backend/...
go install heim/backend/cmd/heim-backend
go install heim/backend/cmd/heimlich

cd client
npm install
PATH=${PATH}:/go/src/heim/client/node_modules/.bin
gulp build

mkdir /hzp
cp /go/bin/heim-backend /hzp
mv /go/src/heim/client/build /hzp/static
cd /hzp
find static -type f | xargs heimlich heim-backend

s3cmd put heim-backend.hzp s3://heim-release/${REVISION}
