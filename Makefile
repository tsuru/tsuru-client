# Copyright 2017 tsuru-client authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Python interpreter path
PYTHON := $(shell which python)

GHR := $(shell which ghr)
GITHUB_TOKEN := $(shell git config --global --get github.token || echo $$GITHUB_TOKEN)

LINTER_ARGS_SLOW = \
	-j 4 --enable-gc -s vendor -e '.*/vendor/.*' --vendor --enable=misspell --enable=gofmt --enable=goimports --enable=unused \
	--disable=dupl --disable=gocyclo --disable=errcheck --disable=golint --disable=interfacer --disable=gas \
	--disable=structcheck --deadline=60m --tests

LINTER_ARGS = \
	$(LINTER_ARGS_SLOW) --disable=staticcheck --disable=unused --disable=gosimple

release:
	@if [ ! $(version) ]; then \
		echo "version parameter is required... use: make release version=<value>"; \
		exit 1; \
	fi
	@if [ "$(GHR)" == "" ]; then \
		echo "ghr is required. Instructions: github.com/tcnksm/ghr"; \
		exit 1; \
	fi
	@if [ ! "$(GITHUB_TOKEN)" ]; then \
		echo "github token should be configurated. Instructions: github.com/tcnksm/ghr"; \
		exit 1; \
	fi

	@echo " ==> Releasing tsuru $(version) version."

	@echo " ==> Replacing version string."
	@sed -i "" "s/version = \".*\"/version = \"$(version)\"/g" tsuru/main.go

	@echo " ==> Building binaries."
	@./misc/build-all.sh

	@echo " ==> Bumping version."
	@git add tsuru/main.go
	@git commit -m "bump to $(version)"

	@echo " ==> Creating tag."

	@git tag $(version)

	@echo " ==> Uploading binaries to github."

	ghr --repository tsuru-client --username tsuru --draft --recreate $(version) dist/

	@echo " ==> Pushing changes to github."

	@git push --tags
	@git push origin master

doc-requirements:
	@go install ./...
	@pip install -r requirements.txt

docs-clean:
	@rm -rf ./docs/build

doc: docs-clean doc-requirements
	@tsuru_sphinx tsuru docs/ && cd docs && make html SPHINXOPTS="-N -W"

docs: doc

docker-test:
	docker run --rm -v ${PWD}:/go/src/github.com/tsuru/tsuru-client -w /go/src/github.com/tsuru/tsuru-client golang:latest sh -c "make test"

test:
	go test ./... -check.v

install:
	go install ./...

build-all:
	./misc/build-all.sh

build:
	go build -o ./bin/tsuru ./tsuru

check-docs: build
	./misc/check-all-cmds-docs.sh

metalint:
	go get -u github.com/alecthomas/gometalinter; \
	gometalinter --install; \
	go install ./...; \
	go test -i ./...; \
	gometalinter $(LINTER_ARGS) ./...; \

.PHONY: doc docs release manpage
