#!/bin/bash
GOARCH=amd64 GOOS=linux go build -o build/server *.go
git subtree push --prefix build dokku main
