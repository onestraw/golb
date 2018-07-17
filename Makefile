PKGS := $(shell go list ./...)
TESTFLAG=-race -cover

test:
	go test $(TESTFLAG) $(PKGS)

test-verbose:
	go test -v $(TESTFLAG) $(PKGS)

build:
	go install github.com/onestraw/golb/cmd/golb/
