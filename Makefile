# Copyright 2017 tsuru-client authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Python interpreter path
PYTHON := $(shell which python)

GHR := $(shell which ghr)
GITHUB_TOKEN := $(shell git config --global --get github.token || echo $$GITHUB_TOKEN)
GIT_TAG_VER := $(shell git describe --tags 2>/dev/null || echo "$${TSURU_BUILD_VERSION:-dev}")

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

metalint:
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin
	go install ./...
	go test -i ./...
	$$(go env GOPATH)/bin/golangci-lint run -c ./.golangci.yml ./...

.PHONY: doc docs release manpage godownloader install-scripts
