FROM ubuntu:14.04

MAINTAINER Max Goodman <c@chromakode.com>

RUN apt-get update && apt-get dist-upgrade -y
RUN apt-get install -y nodejs nodejs-legacy npm

# for phantomjs
RUN apt-get install -y libfontconfig

# install node_modules in /srv/heim/client/node_modules
WORKDIR /srv/heim/client
COPY package.json /srv/heim/client/
RUN npm install; rm package.json

# copy source code to /srv/heim/client/src
WORKDIR /srv/heim/client/src
COPY . /srv/heim/client/src

ENV PATH $PATH:../node_modules/.bin
