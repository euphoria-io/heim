#!/bin/bash

set -ex

go install \
    -ldflags "-X main.version=`git --git-dir=src/euphoria.io/heim/.git rev-parse HEAD`" \
    euphoria.io/heim/heimctl

export PATH=/go/bin:"$PATH"
exec $*
