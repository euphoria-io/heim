FROM heim_backend
MAINTAINER Logan Hanks <logan.hanks@gmail.com>

RUN go install heim/backend/cmd/heim-upgradedb

CMD /go/bin/heim-upgradedb -psql 'postgres://postgres:heimlich@psql/heim?sslmode=disable'
