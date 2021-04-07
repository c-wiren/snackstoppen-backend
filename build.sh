#!/bin/bash
GOARCH=amd64 GOOS=linux go build -ldflags="-w -s" -o build/server *.go
git subtree push --prefix build dokku main
