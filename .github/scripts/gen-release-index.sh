#!/bin/bash
set -euo pipefail

REPOSITORY="$1"

# Generate a simple index.html with a directory listing
cat <<-EOF
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Pacman Builds</title>
	</head>
	<body>
		<h1>Pacman Builds</h1>
		<h2>Available Releases and Assets</h2>
		<ul>
EOF

gh release list \
	--repo "$REPOSITORY" \
	--json tagName --limit 25 \
| jq -r '.[].tagName' \
| while read -r TAG; do
	RELEASE_DATA=$(gh release view "$TAG" \
	  --repo "$REPOSITORY" \
	  --json url,name,body,assets)

	URL=$(jq -r '.url' <<< "$RELEASE_DATA")
	NAME=$(jq -r '.name' <<< "$RELEASE_DATA")
	BODY=$(jq -r '.body' <<< "$RELEASE_DATA")

	# Fallback to tag name if the release title is empty
	DISPLAY_NAME="${NAME:-$TAG}"

	# Start a list item for the release with title and notes
	cat <<-EOF
		<li>
			<a href="$URL">$DISPLAY_NAME</a>
			<p>$BODY</p>
			<ul>
	EOF

	# Extract and list assets as nested list items with download links
	jq -r \
		'.assets[] | "<li><a href=\"\(.url)\">\(.name)</a> (\(.size) bytes)</li>"' \
		<<< "$RELEASE_DATA"

	# Close the nested list for assets
	cat <<-EOF
		  </ul>
		</li>
	EOF
done

cat <<-EOF
		</ul>
	</body>
</html>
EOF
