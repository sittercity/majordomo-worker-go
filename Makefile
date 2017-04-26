APP_NAME=majordomo-worker-go
PACKAGE_SUFFIX=-dev

VERSION=$(shell (date '+%Y%m%d%H%M%S') | tee version.txt)
FILELIST=$(shell cat file.list)
DEB=$(APP_NAME)$(PACKAGE_SUFFIX)_$(VERSION)_amd64.deb

RUNTIME_PACKAGES=libzmq-dev

default: test

vet:
	go vet ./.

test: vet
	@go list -f '{{.Dir}}/test.cov {{.ImportPath}}' ./ \
			| while read coverage package ; do go test -tags test -coverprofile "$$coverage" "$$package" ; done \
			| awk -W interactive '{ print } /^FAIL/ { failures++ } END { exit failures }' ;
	@go list -f '{{.Dir}}/test.cov' ./ \
 			| while read coverage ; do go tool cover -func "$$coverage" ; done \
			| awk '$$3 !~ /^100/ { print; gaps++ } END { exit gaps }' ;

integration-test:

clean:
	rm -rf build
	rm -rf reports

.PHONY: default clean dist integration-test setup test vet
