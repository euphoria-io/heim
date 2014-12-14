FROM ubuntu:14.04

MAINTAINER Max Goodman <c@chromakode.com>

RUN apt-get update && apt-get dist-upgrade -y
RUN apt-get install -y nodejs nodejs-legacy npm

WORKDIR /srv/heim/client

COPY package.json /srv/heim/client/
RUN npm install

COPY . /srv/heim/client

ENV PATH $PATH:node_modules/.bin
RUN gulp build
