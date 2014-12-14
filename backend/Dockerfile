FROM golang
MAINTAINER Logan Hanks <logan.hanks@gmail.com>

ADD . /go/src/heim/backend
RUN go get heim/backend/cmd/heim-backend
RUN go install heim/backend/cmd/heim-backend

# /static should be provided as a volume
ENTRYPOINT /go/bin/heim-backend -static /static -http :80
EXPOSE 80
