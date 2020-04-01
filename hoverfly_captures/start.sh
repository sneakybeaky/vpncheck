#!/usr/bin/env bash

set -e -x

hoverctl start
hoverctl mode simulate
hoverctl import happy.json

PROXY_PORT=`hoverctl config proxy-port`
PROXY_HOST=`hoverctl config host`


cat << EndOfMessage
To start using hoverfly

    export  HTTP_PROXY="http://${PROXY_HOST}:${PROXY_PORT}"

EndOfMessage
