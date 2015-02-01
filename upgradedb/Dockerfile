FROM golang
MAINTAINER Logan Hanks <logan.hanks@gmail.com>

ENV GOPATH /go
ENV PATH $PATH:/go/bin
RUN go get github.com/rubenv/sql-migrate/...
WORKDIR /migrations
