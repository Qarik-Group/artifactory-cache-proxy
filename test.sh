#!/bin/bash

go run main.go -v &
sleep 1

curl --proxy localhost:8080 --proxytunnel 'http://example.com?foo=bar'

kill -9 $(lsof -iTCP:8080 -sTCP:LISTEN | tail -n1 | xargs | cut -d' ' -f2)
