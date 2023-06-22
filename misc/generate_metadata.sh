#!/bin/sh
# Copyright © 2023 tsuru-client authors
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

set -eu
output_file="${1:-}"
[ -z "$output_file" ] && echo "Usage: $0 <FILE>" && exit 1

fail=0
[ -z "${META_PROJECT_NAME:-}" ] && echo "No META_PROJECT_NAME env" && _=$(( fail+=1 ))
[ -z "${META_VERSION:-}" ] && echo "No META_VERSION env" && _=$(( fail+=1 ))
[ -z "${META_TAG:-}" ] && echo "No META_TAG env" && _=$(( fail+=1 ))
# [ -z "${META_PREVIOUS_TAG:-}" ] && echo "No META_PREVIOUS_TAG env" && _=$(( fail+=1 ))
[ -z "${META_COMMIT:-}" ] && echo "No META_COMMIT env" && _=$(( fail+=1 ))
[ -z "${META_DATE:-}" ] && echo "No META_DATE env" && _=$(( fail+=1 ))
[ "$fail" -gt 0 ] && exit 1

echo "Generating metadata for ${output_file}"
cat > "$output_file" <<EOF
{
  "project_name": "${META_PROJECT_NAME}",
  "version": "${META_VERSION}",
  "tag": "${META_TAG}",
  "previous_tag": "${META_PREVIOUS_TAG}",
  "commit": "${META_COMMIT}",
  "date": "${META_DATE}"
}
EOF
