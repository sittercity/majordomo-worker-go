APP_NAME=majordomo-worker-go
PACKAGE_SUFFIX=-dev

VERSION=$(shell (date '+%Y%m%d%H%M%S') | tee version.txt)
FILELIST=$(shell cat file.list)
DEB=$(APP_NAME)$(PACKAGE_SUFFIX)_$(VERSION)_amd64.deb

RUNTIME_PACKAGES=sc-libzmq4

default: test

vet:
	go vet ./...

test: vet
	mkdir -p reports/cov
	go test -tags test -coverprofile reports/main.cov -v . > reports/main.txt
	[ ! -f reports/main.cov ] || gocov convert reports/main.cov | gocov-html > reports/cov/main.html

integration-test:

clean:
	rm -rf build
	rm -rf reports

.PHONY: default clean dist integration-test build/chime-public-api-majordomo-worker setup test vet
