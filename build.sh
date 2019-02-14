#!/bin/bash
export GO111MODULE=on
export GOFLAGS=-mod=vendor
GOOS=linux GOARCH=amd64 go build -o proxy-linux main.go
