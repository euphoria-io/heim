FROM ubuntu:16.04
MAINTAINER Logan Hanks <logan@euphoria.io>

ARG HEIM_BRANCH
ARG HEIM_COMMIT

RUN apt-get update && apt-get dist-upgrade -y
RUN apt-get install -y nodejs nodejs-legacy npm git golang-1.10

# for phantomjs
RUN apt-get install -y libfontconfig

ENV PATH /root/go/src/euphoria.io/heim/client/node_modules/.bin:/usr/lib/go-1.10/bin:$PATH

# copy source code to /srv/heim/client/src
ADD ./ /root/go/src/euphoria.io/heim/

# install client dependencies
WORKDIR /root/go/src/euphoria.io/heim/client
RUN npm install

# build client
ENV NODE_ENV production
RUN gulp build

# build server
RUN go install -ldflags "-X main.version=${HEIM_COMMIT}" euphoria.io/heim/heimctl
RUN go install euphoria.io/heim/heimlich
