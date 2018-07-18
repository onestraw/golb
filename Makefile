PKGS := $(shell go list ./...)
TESTFLAG=-race -cover

test:
	GOCACHE=off go test $(TESTFLAG) $(PKGS)

test-verbose:
	GOCACHE=off go test -v $(TESTFLAG) $(PKGS)

build:
	go install github.com/onestraw/golb/cmd/golb/
