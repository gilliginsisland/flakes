# help-book

Generate a static HTML help book from a Markdown file. The Go executable includes
its page template and styles; it needs no browser JavaScript or Node.js.

## Usage

```sh
nix run .#help-book -- --title "Example Help" guide.md ./html
```

Run the Nix command from the repository root. Or build and run with Go from this
directory:

```sh
go build .
./help-book --title "Example Help" guide.md ./html
```

The first argument is the Markdown source file, and the second is the output
directory. Use a fresh output directory when generating a book after renaming or
removing sections.

- `--title` sets the sidebar and book title (default: `Help`).
- `--book-id` optionally sets the index page's `AppleTitle` metadata for Apple Help.

Level-two headings become sections, and level-three headings become their child
pages. Deeper headings stay within their page. The level-one document heading is
omitted in favor of the book title. An index, sidebar, breadcrumbs, and section
contents link the pages together. Page filenames use lowercase heading text with
non-alphanumeric runs replaced by hyphens. Section and page names must produce
unique filenames; `index` is reserved.

GitHub-flavored Markdown includes tables and fenced code blocks. Raw HTML is
preserved, including explicit anchors, so only process documentation you trust.
Use links such as `rules.html` or `proxies.html#timeout` between generated pages.

The output is ordinary HTML and CSS. Preview it with a local static HTTP server,
or package it into an Apple Help book. Platform-specific bundle metadata and
search indexing remain the caller's responsibility.

This directory is a separate Go module with vendored dependencies. The top-level
flake discovers `package.nix` and exposes the `help-book` package. Go and Nix builds
use the checked-in `vendor` directory without downloading Go dependencies.
