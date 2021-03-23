#!/bin/bash
GOARCH=amd64 GOOS=linux go build -o app/app *.go
subtree push --prefix build dokku-backend main