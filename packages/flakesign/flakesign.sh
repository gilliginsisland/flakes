set -euo pipefail

usage() {
	echo "usage: flakesign installable artifact.tgz [cert.pem]" >&2
}

if (( $# < 2 || $# > 3 )); then
	usage
	exit 2
fi

installable="$1"
artifact_path="$2"
cert_arg="${3:-}"

package_dir="$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/flakesign.XXXXXX")"
cert_path="$package_dir/cert.pem"

cleanup() {
	rm -rf "$package_dir"
}
trap cleanup EXIT

if [[ -n "$cert_arg" ]]; then
	if [[ ! -f "$cert_arg" ]]; then
		echo "Certificate file not found: $cert_arg" >&2
		exit 1
	fi
	cp "$cert_arg" "$cert_path"
elif [[ -n "${FLAKESIGN_CERT:-}" ]]; then
	printf '%s' "$FLAKESIGN_CERT" > "$cert_path"
else
	echo "Pass a PEM certificate file or set FLAKESIGN_CERT." >&2
	exit 1
fi
chmod 600 "$cert_path"

result_dir="$(
	nix --extra-experimental-features 'nix-command flakes' \
		build --show-trace -L --no-link --print-out-paths "$installable"
)"

if [[ -z "$result_dir" || "$result_dir" == *$'\n'* ]]; then
	echo "Expected exactly one output path from: $installable" >&2
	printf '%s\n' "$result_dir" >&2
	exit 1
fi

cp -R "$result_dir/." "$package_dir"
chmod -R u+w "$package_dir"

apps_file="$package_dir/.flakesign-apps"

find "$package_dir" -type d -name '*.app' -print |
	while IFS= read -r app; do
		case "$app" in
			*.app/*) ;;
			*) printf '%s\n' "$app" ;;
		esac
	done > "$apps_file"

if [[ ! -s "$apps_file" ]]; then
	echo "No top-level .app bundles found in $result_dir." >&2
	exit 1
fi

signed=0
while IFS= read -r app; do
	rcodesign sign \
		--timestamp-url none \
		--pem-file "$cert_path" \
		"$app"
	signed=$((signed + 1))
done < "$apps_file"

verified=0
while IFS= read -r app; do
	while IFS= read -r path; do
		if [[ "$(file "$path")" == *Mach-O* ]]; then
			if ! output="$(rcodesign verify "$path" 2>&1)"; then
				printf '%s\n' "$output" >&2
				exit 1
			fi
			verified=$((verified + 1))
		fi
	done < <(find "$app" -type f -print)
done < "$apps_file"

echo "Signed $signed app bundle(s)."
echo "Verified $verified Mach-O signature(s)."

rm -f "$apps_file" "$cert_path"
tar -czvf "$artifact_path" -C "$package_dir" .
