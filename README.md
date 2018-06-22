# Go LB
[![Build Status](https://travis-ci.org/onestraw/golb.svg?branch=master)](https://travis-ci.org/onestraw/golb)
[![Coverage Status](https://coveralls.io/repos/github/onestraw/golb/badge.svg?branch=master)](https://coveralls.io/github/onestraw/golb?branch=master)
[![godoc](https://godoc.org/github.com/onestraw/golb?status.svg)](https://godoc.org/github.com/onestraw/golb)

A load balancer with features:

- [roundrobin](roundrobin/): smooth weighted roundrobin method
- [chash](chash/): cosistent hashing method 
- multiple LB instances, start/stop at runtime
- dynamic configuration
- passive health check
- stats
