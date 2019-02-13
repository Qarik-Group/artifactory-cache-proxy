#!/bin/bash

go run main.go -v &
sleep 1

# curl -k --proxy localhost:8080 --proxytunnel 'https://example.com?foo=bar'
# curl -k --proxy localhost:8080 'https://example.com?foo=baz'
curl -k --proxy localhost:8080 -o release.tgz 'https://bosh.io/d/github.com/cloudfoundry-community/cron-boshrelease?v=1.1.3'

sleep 5

kill -9 $(lsof -iTCP:8080 -sTCP:LISTEN | tail -n1 | xargs | cut -d' ' -f2)
