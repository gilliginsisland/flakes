#!/bin/bash
set -euo pipefail

gen_jobs(){
	nix eval --json .#actions.matrix | jq -c '.[]' | while read -r line; do
		TAG=$(jq -r '"\(.app)@v\(.version)"' <<< "$line")
		if ! git ls-remote --tags --exit-code origin "refs/tags/${TAG}" > /dev/null 2>&1; then
			echo "$line"
		fi
	done
}
JOBS=$(gen_jobs)

jq -src \
	'"build="+(. | @json)' \
	<<< "$JOBS"

jq -src \
	'"release="+(group_by(.app,.version) | map(.[0] | {app,version,changes,tag:"\(.app)@v\(.version)"}) | @json)' \
	<<< "$JOBS"

jq -src \
	'"has_jobs="+(length > 0 | @json)' \
	<<< "$JOBS"
