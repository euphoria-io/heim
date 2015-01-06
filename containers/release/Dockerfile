FROM ubuntu:14.04
MAINTAINER Logan Hanks <logan.hanks@gmail.com>

RUN apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y --force-yes \
        ca-certificates curl gcc libc6-dev make bzr git mercurial \ 
        nodejs nodejs-legacy npm libfontconfig \
        s3cmd
    && rm -rf /var/lib/apt/lists/*

ENV GOLANG_VERSION 1.4

RUN curl -sSL https://golang.org/dl/go$GOLANG_VERSION.src.tar.gz | tar -v -C /usr/src -xz
RUN cd /usr/src/go/src && ./make.bash --no-clean 2>&1

ENV PATH /usr/src/go/bin:$PATH

RUN mkdir -p /go/src
ENV GOPATH /go
ENV PATH /go/bin:$PATH

ENV REVISION 9d8d272

RUN mkdir /release
ADD s3cfg /root/.s3cfg
ADD build.sh /
CMD /build.sh
