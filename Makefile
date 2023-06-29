# Copyright © 2023 tsuru-client authors
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

GOCMD	?= go
PARALL	?= $(shell { nproc --all || echo 1 ; } | xargs -I{} expr {} / 2 + 1 )
GOTEST	?= $(GOCMD) test -timeout 10s -parallel $(PARALL)
GOVET	?= $(GOCMD) vet
GOFMT	?= gofmt
BINARY	?= tsuru
TSURUGO	?= $(shell $(GOCMD) env GOPATH)/bin/$(BINARY)
VERSION	?= $(shell git describe --tags --dirty --match='v*' 2> /dev/null || echo dev)
COMMIT	?= $(shell git rev-parse --short HEAD 2> /dev/null || echo "")
DATEUTC	?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
FILES	?= $(shell find . -type f -name '*.go')

GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: all test build coverage contributors scripts

default: help

## Build:
build: ## Build your project and put the output binary in build/
	$(GOCMD) build -ldflags "-s -w -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.dateStr=$(DATEUTC)'" -o build/$(BINARY) ./tsuru

install: build  ## Build your project and install the binary in $GOPATH/bin/
	rm -f $(TSURUGO)
	cp build/$(BINARY) $(TSURUGO)

clean: ## Remove build related file
	rm -fr ./build
	rm -fr ./coverage

clean-all: ## Remove build related file and installed binary
	rm -fr ./build
	rm -fr ./coverage
	rm -f $(TSURUGO)

fmt: ## Format your code with gofmt
	$(GOFMT) -w .

addlicense: ## Add licence header to all files
ifeq (, $(shell which addlicense))
	go install github.com/google/addlicense@latest
endif
	addlicense -f LICENSE-HEADER .

contributors: ## Update CONTRIBUTORS file
	./scripts/contributors_file.sh > CONTRIBUTORS

## Test:
test: ## Run the tests of the project (fastest)
	$(GOVET) ./...
	$(GOTEST) ./...

test-ci: ## Run ALL the tests of the project (+race)
	$(GOVET) ./...
	$(GOTEST) -v -race ./...

test-coverage: ## Run the tests of the project and export the coverage
	rm -fr coverage && mkdir coverage
	$(GOTEST) -cover -covermode=atomic -coverprofile=coverage/coverage.out ./...
	@echo ""
	$(GOCMD) tool cover -func=coverage/coverage.out
	@echo ""
	$(GOCMD) tool cover -func=coverage/coverage.out -o coverage/coverage.txt
	$(GOCMD) tool cover -html=coverage/coverage.out -o coverage/coverage.html

coverage: test-coverage  ## Run test-coverage and open coverage in your browser
	$(GOCMD) tool cover -html=coverage/coverage.out

## Lint:
lint: lint-license-header lint-go check-contributors ## Run all available linters

lint-go: ## Use gofmt and staticcheck on your project
ifneq (, $(shell $(GOFMT) -l . ))
	@echo "This files are not gofmt compliant:"
	@$(GOFMT) -l .
	@echo "Please run 'make fmt' to format your code"
	@exit 1
endif
ifeq (, $(shell which staticcheck))
	go install honnef.co/go/tools/cmd/staticcheck@latest
endif
	staticcheck ./...

lint-license-header: ## Check if all files have the license header
ifeq (, $(shell which addlicense))
	go install github.com/google/addlicense@latest
endif
	@echo "addlicense -check -f LICENSE-HEADER -ignore coverage/** ."
	@addlicense -check -f LICENSE-HEADER -ignore coverage/** . \
		|| (echo "Some files are missing the license header, please run '$(CYAN)make addlicense$(RESET)' to add it" && exit 1)

check-contributors: ## Check if all contributors are listed on the CONTRIBUTORS file
	@echo "check CONTRIBUTORS"
ifneq (, $(shell ./scripts/contributors_file.sh | diff CONTRIBUTORS - ))
	./scripts/contributors_file.sh | diff CONTRIBUTORS -
	@echo "Some contributors are missing from the CONTRIBUTORS file, please run '$(CYAN)make contributors$(RESET)' to add them" && exit 1
endif

## Help:
help: ## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)

env:    ## Print useful environment variables to stdout
	@echo '$$(GOCMD)   :' $(GOCMD)
	@echo '$$(GOTEST)  :' $(GOTEST)
	@echo '$$(GOVET)   :' $(GOVET)
	@echo '$$(BINARY)  :' $(BINARY)
	@echo '$$(VERSION) :' $(VERSION)
	@echo '$$(COMMIT)  :' $(COMMIT)
	@echo '$$(DATEUTC) :' $(DATEUTC)
	@echo '$$(FILES#)  :' $(shell echo $(FILES) | wc -w)
