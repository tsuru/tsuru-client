# Copyright 2015 tsuru-client authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Python interpreter path
PYTHON := $(shell which python)


release:
	@if [ ! $(version) ]; then \
		echo "version parameter is required... use: make release version=<value>"; \
		exit 1; \
	fi

	@echo "Releasing tsuru $(version) version."

	@echo "Replacing version string."
	@sed -i "" "s/version = \".*\"/version = \"$(version)\"/g" tsuru/main.go

	@git add tsuru/main.go
	@git commit -m "bump to $(version)"

	@echo "Creating $(version) tag."
	@git tag $(version)

	@git push --tags
	@git push origin master

	@echo "$(version) released!"

requirements: requirements.txt
	@pip install -r $<

docs-clean:
	@rm -rf ./docs/build

docs-requirements:
	@pip install -r docs/requirements.txt

doc: docs-clean docs-requirements
	@tsuru_sphinx tsuru docs/ && cd docs && make html SPHINXOPTS="-N -W"

docs: doc

manpage: docs docs/source/exts/man_pages.py
	$(PYTHON) $(word 2, $^)

.PHONY: doc docs release manpage
