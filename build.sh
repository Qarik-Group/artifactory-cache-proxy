#!/bin/bash

GOOS=linux GOARCH=amd64 go build -o proxy-linux main.go
