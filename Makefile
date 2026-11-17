# Copyright 2017 tsuru-client authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Python interpreter path
PYTHON := $(shell which python)

GIT_TAG_VER := $(shell git describe --tags 2>/dev/null || echo "$${TSURU_BUILD_VERSION:-dev}")

doc-requirements: install
	@pip install -r requirements.txt

docs-clean:
	@rm -rf ./docs/build

doc: docs-clean doc-requirements
	@tsuru_sphinx tsuru docs/ && cd docs && make html SPHINXOPTS="-N -W"

docs: doc

docker-test:
	docker run --rm -v ${PWD}:/go/src/github.com/tsuru/tsuru-client -w /go/src/github.com/tsuru/tsuru-client golang:latest sh -c "make test"

test:
	go test -race ./... -check.v

install:
	go install ./...

build-all:
	./misc/build-all.sh

build:
	go build -ldflags "-s -w -X 'main.version=$(GIT_TAG_VER)'" -o ./bin/tsuru ./tsuru

check-docs: build
	./misc/check-all-cmds-docs.sh

godownloader:
	git clone https://github.com/goreleaser/godownloader.git /tmp/godownloader
	cd /tmp/godownloader && go install .
	rm -rf /tmp/godownloader

install.sh: .goreleaser.yml godownloader
	godownloader --repo tsuru/tsuru-client $< >$@

install-scripts: install.sh

metalint: install
ifeq (, $(shell which golangci-lint))
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.12.2
endif
	$$(go env GOPATH)/bin/golangci-lint run -c ./.golangci.yml ./...

yamlfmt: ## Format your code with yamlfmt
ifeq (, $(shell which yamlfmt))
	go install github.com/google/yamlfmt/cmd/yamlfmt@v0.9.0
endif
	yamlfmt .

yamllint: ## Check the yaml is valid and correctly formatted
ifeq (, $(shell which yamlfmt))
	go install github.com/google/yamlfmt/cmd/yamlfmt@v0.9.0
endif
	@echo "yamlfmt --quiet --lint ."
	@yamlfmt --quiet --lint . \
		|| ( echo "Please run 'make yamlfmt' to fix it (if a format error)" && exit 1 )


.PHONY: doc docs manpage godownloader install-scripts
