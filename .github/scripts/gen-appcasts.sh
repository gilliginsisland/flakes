#!/bin/bash
set -euo pipefail

REPOSITORY="$1"
APPS=$(jq -c '.[]' <<< "$2")
OUTPUT_DIR="$3"

# Function to generate appcast RSS feed to stdout
generate_appcast_feed() {
	local ENTRY APP SYSTEM VERSION TAG RELEASE
	read -r ENTRY
	APP=$(jq -r '.app' <<< "$ENTRY")
	SYSTEM=$(jq -r '.system' <<< "$ENTRY")
	VERSION=$(jq -r '.version' <<< "$ENTRY")
	# CHANGES=$(jq -r '.changes | .[] | "* "+.' <<< "$ENTRY")
	TAG=$(jq -r '.tag' <<< "$ENTRY")

	echo "Generating appcast for $APP on $SYSTEM (tag: $TAG)..." >&2

	# Fetch release data for the specific tag
	if ! RELEASE=$(gh release view "$TAG" \
		--repo "$REPOSITORY" \
		--json name,body,assets,url,publishedAt)
	then
		echo "Error: Release for tag $TAG not found for $APP on $SYSTEM. Skipping appcast generation." >&2
		return 1
	fi

	local NAME BODY LINK PUBLISHED_AT PUB_DATE
	NAME=$(jq -r '.name' <<< "$RELEASE")
	BODY=$(jq -r '.body' <<< "$RELEASE")
	LINK=$(jq -r '.url' <<< "$RELEASE")
	PUBLISHED_AT=$(jq -r '.publishedAt' <<< "$RELEASE")
	# Convert ISO 8601 date (e.g., 2026-02-18T17:09:59Z) to RFC 2822 format (e.g., Wed, 18 Feb 2026 17:09:59 +0000) for RSS pubDate
	# Handle both Linux (GNU date) and macOS (BSD date)
	if [[ "$OSTYPE" == "darwin"* ]]; then
		# macOS (BSD date) - use -j and -f to parse the date string
		PUB_DATE=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$PUBLISHED_AT" "+%a, %d %b %Y %H:%M:%S %z")
	else
		# Linux (GNU date) - use -d to parse the date string
		PUB_DATE=$(date -R -d "$PUBLISHED_AT")
	fi

	# Extract the specific asset for this system selecting the first match
	local ASSET URL SIZE
	ASSET=$(jq -c --arg system "$SYSTEM" \
		'.assets | map(select(.name | contains($system))) | .[0]' \
		<<< "$RELEASE")
	URL=$(jq -r '.url' <<< "$ASSET")
	SIZE=$(jq -r '.size' <<< "$ASSET")

	cat <<-EOF
		<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:sparkle="http://www.andymatuschak.org/xml-namespaces/sparkle">
			<channel>
				<title>$NAME changelog</title>
				<description>Most recent changes with links to updates.</description>
				<language>en</language>
				<item>
					<title>Version $VERSION</title>
					<link>$LINK</link>
					<sparkle:version>$VERSION</sparkle:version>
					<sparkle:shortVersionString>$VERSION</sparkle:shortVersionString>
					<description><![CDATA[$BODY]]></description>
					<pubDate>$PUB_DATE</pubDate>
					<enclosure url="$URL" length="$SIZE" type="application/octet-stream"/>
				</item>
			</channel>
		</rss>
	EOF
}

# Read JSON lines from stdin (each line is an app-system pair)
while read -r ENTRY; do
	APP=$(jq -r '.app' <<< "$ENTRY")
	SYSTEM=$(jq -r '.system' <<< "$ENTRY")
	APPCAST_FILE="$OUTPUT_DIR/$APP-$SYSTEM-appcast.xml"
	generate_appcast_feed > "$APPCAST_FILE" <<< "$ENTRY"
done <<< "$APPS"
