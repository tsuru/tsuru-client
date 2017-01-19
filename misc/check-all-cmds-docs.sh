#!/bin/bash

# Copyright 2017 tsuru authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

for c in `./bin/tsuru | grep "Available commands" -A 500 | cut -f3 -d' ' | sort -u`
do
    cat ./docs/source/reference.rst | grep "$c" >/dev/null 2>&1
    RESULT=$?
    if [ $RESULT -eq 1 ]
    then
        echo "${c} is not documented"
    fi
done
