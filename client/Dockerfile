FROM ubuntu:14.04
MAINTAINER Max Goodman <max@euphoria.io>

RUN apt-get update && apt-get dist-upgrade -y
RUN apt-get install -y nodejs nodejs-legacy npm git

# for phantomjs
RUN apt-get install -y libfontconfig

# copy source code to /srv/heim/client/src
WORKDIR /srv/heim/client/

ENV PATH $PATH:node_modules/.bin

VOLUME /srv/heim/client/build
