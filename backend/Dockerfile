FROM golang
MAINTAINER Logan Hanks <logan.hanks@gmail.com>

ADD . /go/src/heim/backend
RUN go get heim/backend/cmd/heim-backend
RUN go install heim/backend/cmd/heim-backend

# /go/src/hiem/backend/static should be provided as a volume
# psql host should be provided as a linked container
CMD /go/bin/heim-backend -static /go/src/heim/backend/static -http :80 -psql 'postgres://postgres:heimlich@psql/heim?sslmode=disable'

EXPOSE 80
