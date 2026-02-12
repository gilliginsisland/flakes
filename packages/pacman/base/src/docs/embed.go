package docs

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed README.md
var readme string
var readmeJSON, _ = json.Marshal(readme)

//go:embed github-markdown.css
var githubMarkdownCSS string

//go:embed marked.umd.js
var markedJS string

var HTML = fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Documentation</title>
	<style>
		html,
		body {
			padding: 15px;
			margin: 0;
			overscroll-behavior: none;
		}
	</style>
	<style>%s</style>
</head>
<body class="markdown-body">
	<div id="content"></div>
	<script>%s</script>
	<script>
		document.getElementById('content').outerHTML = marked.parse(%s);
	</script>
</body>
</html>`, githubMarkdownCSS, markedJS, readmeJSON)
