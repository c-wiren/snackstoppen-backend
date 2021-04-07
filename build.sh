#!/bin/bash
GOARCH=amd64 GOOS=linux go build -o app/app *.go
git subtree push --prefix build dokku main
