FROM ubuntu:16.04
MAINTAINER Logan Hanks <logan@euphoria.io>

ENV PATH /root/go/src/euphoria.io/heim/client/node_modules/.bin:/usr/lib/go-1.10/bin:$PATH

# install bazel and upgrade OS
RUN apt-get update && apt-get dist-upgrade -y
RUN apt-get install -y openjdk-8-jdk curl
RUN echo "deb [arch=amd64] http://storage.googleapis.com/bazel-apt stable jdk1.8" > /etc/apt/sources.list.d/bazel.list
RUN curl https://bazel.build/bazel-release.pub.gpg | apt-key add -
RUN apt-get update
RUN apt-get install -y bazel

# install node and npm
RUN apt-get install -y nodejs nodejs-legacy npm

# install golang
RUN apt-get install -y git golang-1.10

# install phantomjs dependency
RUN apt-get install -y libfontconfig

# copy source code to /srv/heim/client/src
ADD ./ /root/go/src/euphoria.io/heim/

# install client dependencies
WORKDIR /root/go/src/euphoria.io/heim/client
RUN npm install

# build client
ENV NODE_ENV production
RUN gulp build

# build server
RUN bazel build --workspace_status_command=./bzl/status.sh //heimctl
