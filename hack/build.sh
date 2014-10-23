#!/bin/bash -xe

basedir=`readlink -f $(dirname "${BASH_SOURCE[0]}")/..`

pushd $basedir >/dev/null
mkdir -p bin
rm -rf bin/*
go build -o bin/git-sync cmd/sync/main.go
popd >/dev/null
