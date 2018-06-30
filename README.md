# Go LB
[![Build Status](https://travis-ci.org/onestraw/golb.svg?branch=master)](https://travis-ci.org/onestraw/golb)
[![Coverage Status](https://coveralls.io/repos/github/onestraw/golb/badge.svg?branch=master)](https://coveralls.io/github/onestraw/golb?branch=master)
[![godoc](https://godoc.org/github.com/onestraw/golb?status.svg)](https://godoc.org/github.com/onestraw/golb)

A load balancer with features:

- [roundrobin](roundrobin/): smooth weighted roundrobin method
- [chash](chash/): cosistent hashing method
- [balancer](balancer/): multiple LB instances, passive health check
- [controller](controller/): dynamic configuration, REST API to start/stop/add/remove LB at runtime
- [statistics](stats/): HTTP method/path/code/bytes

## Let's try

- Terminal #1: make build && golb -config=golb.json
- Terminal #2: watch 'curl -H "Host:localhost" http://127.0.0.1:8081 > /dev/null'
- Terminal #3: python -m SimpleHTTPServer 10001 & python -m SimpleHTTPServer 10002 & python -m SimpleHTTPServer 10005
- Terminal #4:

#### test basic stats

    curl -u admin:admin http://127.0.0.1:6587/stats
    curl -u admin:admin http://127.0.0.1:6587/vs
    curl -u admin:admin http://127.0.0.1:6587/vs/web

#### test add/remove pool member

    curl -XPOST -u admin:admin -d '{"address":"127.0.0.1:10005"}' http://127.0.0.1:6587/vs/web/pool
    curl -u admin:admin http://127.0.0.1:6587/vs/web
    curl -u admin:admin http://127.0.0.1:6587/stats

    curl -XDELETE -u admin:admin -d '{"address":"127.0.0.1:10005"}' http://127.0.0.1:6587/vs/web/pool

#### test enable/disable LB instance

    curl -XPOST -u admin:admin -d '{"action":"disable"}' http://127.0.0.1:6587/vs/web
    curl -u admin:admin http://127.0.0.1:6587/vs/web
    curl -XPOST -u admin:admin -d '{"action":"enable"}' http://127.0.0.1:6587/vs/web
