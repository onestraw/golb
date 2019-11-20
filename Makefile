PKGS := $(shell go list ./... | grep -v 'examples/')
TXT_FILES := $(shell find * -type f -not -path 'vendor/**' -not -name 'go.sum')
TESTFLAG=-race -cover
export GO111MODULE=on

test:
	go test $(TESTFLAG) $(PKGS)

test-verbose:
	go test -v $(TESTFLAG) $(PKGS)

loadtest:
	dd if=/dev/zero ibs=1k count=1 of=test.data
	ab -k -c100 -t30 -T application/octet-stream -p test.data 'http://127.0.0.1:8081/'
	rm test.data

check: vet lint misspell staticcheck

lint:
	@echo "golint"
	golint -set_exit_status $(PKGS)

vet:
	@echo "vet"
	go vet $(PKGS)

misspell:
	@echo "misspell"
	misspell -source=text -error $(TXT_FILES)

staticcheck:
	@echo "staticcheck"
	staticcheck $(PKGS)

build:
	go install github.com/onestraw/golb/cmd/golb/
