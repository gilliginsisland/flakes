#!/bin/bash
set -euo pipefail

MATRIX=$(nix eval --json .#actions.matrix | jq -c 'map(.tag = "\(.app)@v\(.version)")')

filter_jobs(){
	while read -r line; do
		TAG=$(jq -r '.tag' <<< "$line")
		if ! git ls-remote --tags --exit-code origin "refs/tags/${TAG}" > /dev/null 2>&1; then
			echo "$line"
		fi
	done
}
JOBS=$(jq -c '.[]' <<< "$MATRIX" | filter_jobs)

echo "apps=$MATRIX"

jq -src \
	'"build="+(map({app,system,runner,output}) | @json)' \
	<<< "$JOBS"

jq -src \
	'"release="+(group_by(.app,.version) | map(.[0] | {app,version,changes,tag}) | @json)' \
	<<< "$JOBS"

jq -src \
	'"has_jobs="+(length > 0 | @json)' \
	<<< "$JOBS"
