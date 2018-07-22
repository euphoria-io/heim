#!/bin/bash
set -ex

echo "deb [arch=amd64] http://storage.googleapis.com/bazel-apt stable jdk1.8" | sudo tee /etc/apt/sources.list.d/bazel.list
curl --silent https://bazel.build/bazel-release.pub.gpg | sudo apt-key add -
sudo apt-get update -o Dir::Etc::sourcelist="sources.list.d/bazel.list" \
    -o Dir::Etc::sourceparts="-" \
    -o APT::Get::List-Cleanup="0"
sudo apt-get install openjdk-8-jdk bazel

bazel test //...
