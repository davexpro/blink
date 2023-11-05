#!/usr/bin/env bash

rm -rf output/
mkdir -p output/

GIT_SHA=$(git rev-parse --short HEAD || echo "GitNotFound")
DATE=$(date +'%F %T %z')

go build -ldflags "-s -w -extldflags '-static' -X main.magic=${GIT_SHA} -X 'main.date=${DATE}'" -o output/blink-cli cmd/client/main.go
go build -ldflags "-s -w -extldflags '-static' -X main.magic=${GIT_SHA} -X 'main.date=${DATE}'" -o output/blink-srv cmd/server/main.go
