# Go LB
[![Build Status](https://travis-ci.org/onestraw/golb.svg?branch=master)](https://travis-ci.org/onestraw/golb)
[![Coverage Status](https://coveralls.io/repos/github/onestraw/golb/badge.svg?branch=master)](https://coveralls.io/github/onestraw/golb?branch=master)
[![godoc](https://godoc.org/github.com/onestraw/golb?status.svg)](https://godoc.org/github.com/onestraw/golb)

Yet another load balancer
![golb](golb.png)

## Features

- [roundrobin](roundrobin/): smooth weighted roundrobin method
- [chash](chash/): cosistent hashing method
- [balancer](balancer/): **multiple LB instances, passive health check, SSL offloading**
- [controller](controller/): dynamic configuration, **REST API to start/stop/add/remove LB at runtime**
- [statistics](stats/): HTTP method/path/code/bytes

## Let's try

### configuration

The configurations specify that controller listens on `127.0.0.1:6587`,
and define a virtual server including two servers with the default roundrobin method
```json
{
  "controller": {
      "address": "127.0.0.1:6587",
      "auth": {
          "username": "admin",
          "password": "admin"
      }
  },
  "virtual_server": [
    {
      "name": "web",
      "address": "127.0.0.1:8081",
      "pool": [
        {
          "address": "127.0.0.1:10001",
          "weight": 1
        },
        {
          "address": "127.0.0.1:10002",
          "weight": 2
        }
      ]
    }
  ]
}
```

### run the demo

`make demo`

### query basic stats

    curl -u admin:admin http://127.0.0.1:6587/stats
    curl -u admin:admin http://127.0.0.1:6587/vs
    curl -u admin:admin http://127.0.0.1:6587/vs/web

### add/remove pool member

    curl -XPOST -u admin:admin -d '{"address":"127.0.0.1:10003"}' http://127.0.0.1:6587/vs/web/pool
    curl -u admin:admin http://127.0.0.1:6587/vs/web
    curl -u admin:admin http://127.0.0.1:6587/stats

    curl -XDELETE -u admin:admin -d '{"address":"127.0.0.1:10003"}' http://127.0.0.1:6587/vs/web/pool

### enable/disable LB instance

    curl -XPOST -u admin:admin -d '{"action":"disable"}' http://127.0.0.1:6587/vs/web
    curl -u admin:admin http://127.0.0.1:6587/vs/web
    curl -XPOST -u admin:admin -d '{"action":"enable"}' http://127.0.0.1:6587/vs/web
