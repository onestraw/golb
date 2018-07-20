PKGS := $(shell go list ./...)
TESTFLAG=-race -cover

test:
	GOCACHE=off go test $(TESTFLAG) $(PKGS)

test-verbose:
	GOCACHE=off go test -v $(TESTFLAG) $(PKGS)

loadtest:
	dd if=/dev/zero ibs=1k count=1 of=test.data
	ab -k -c100 -t30 -T application/octet-stream -p test.data 'http://127.0.0.1:8081/'
	rm test.data

build:
	go install github.com/onestraw/golb/cmd/golb/
