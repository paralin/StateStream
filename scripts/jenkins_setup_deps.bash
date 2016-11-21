#!/bin/bash
set +x
set -e

source ./jenkins_scripts/jenkins_env.bash

mkdir -p ./goworkspace/bin
mkdir -p ./goworkspace/src/github.com/fuserobotics
ln -fs $(pwd) ./goworkspace/src/github.com/fuserobotics/statestream

pushd ./goworkspace/src/github.com/fuserobotics/statestream
go get -d -v ./
popd

set -x
