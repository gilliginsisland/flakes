package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	renderhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

//go:embed help.css github-markdown.css page.html
var assets embed.FS

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

var pageTemplate = template.Must(template.New("page.html").Funcs(template.FuncMap{
	"contents": contents,
}).ParseFS(assets, "page.html"))

type topic struct {
	Title    string
	Slug     string
	Depth    int
	Parent   *topic
	Children []*topic
	body     bytes.Buffer
}

type page struct {
	Title     string
	BookTitle string
	BookID    string
	Slug      string
	Parent    *topic
	Groups    []*topic
	Body      template.HTML
	Children  []*topic
}

func contents(topics []*topic, active string) template.HTML {
	var out strings.Builder
	out.WriteString("<ul>")
	for _, t := range topics {
		fmt.Fprintf(&out, `<li><a href="%s.html"`, html.EscapeString(t.Slug))
		if t.Slug == active {
			out.WriteString(` aria-current="page"`)
		}
		fmt.Fprintf(&out, ">%s</a>", html.EscapeString(t.Title))
		if len(t.Children) > 0 {
			out.WriteString(string(contents(t.Children, active)))
		}
		out.WriteString("</li>\n")
	}
	out.WriteString("</ul>")
	return template.HTML(out.String())
}

func generate(source []byte, destination, title, bookID string) error {
	// The input is author-supplied documentation; preserve its HTML anchors.
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(renderhtml.WithUnsafe()),
	)
	document := md.Parser().Parse(text.NewReader(source))
	var introduction bytes.Buffer
	var topics, groups []*topic
	var current, group *topic
	seen := map[string]bool{"index": true}
	for node := document.FirstChild(); node != nil; node = node.NextSibling() {
		if heading, ok := node.(*ast.Heading); ok {
			if heading.Level == 1 {
				continue
			}
			if heading.Level <= 3 {
				name := string(heading.Text(source))
				slug := strings.Trim(slugPattern.ReplaceAllString(strings.ToLower(name), "-"), "-")
				if slug == "" || seen[slug] {
					return fmt.Errorf("heading %q has an empty or duplicate page name %q", name, slug)
				}
				seen[slug] = true
				current = &topic{Title: name, Slug: slug, Depth: heading.Level}
				topics = append(topics, current)
				if heading.Level == 2 {
					group = current
					groups = append(groups, current)
				} else {
					if group == nil {
						return fmt.Errorf("heading %q needs a preceding level-two heading", name)
					}
					current.Parent = group
					group.Children = append(group.Children, current)
				}
				continue
			}
			if current != nil {
				heading.Level -= current.Depth - 1
			}
		}
		body := &introduction
		if current != nil {
			body = &current.body
		}
		if err := md.Renderer().Render(body, source, node); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"help.css", "github-markdown.css"} {
		data, err := assets.ReadFile(name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(destination, name), data, 0o644); err != nil {
			return err
		}
	}
	index := page{
		Title: "Contents", BookTitle: title, BookID: bookID, Slug: "index",
		Groups: groups, Body: template.HTML(introduction.String()), Children: groups,
	}
	if err := writePage(destination, index); err != nil {
		return err
	}
	for _, t := range topics {
		if err := writePage(destination, page{
			Title: t.Title, BookTitle: title, Slug: t.Slug, Parent: t.Parent,
			Groups: groups, Body: template.HTML(t.body.String()), Children: t.Children,
		}); err != nil {
			return err
		}
	}
	return nil
}

func writePage(destination string, p page) error {
	var out bytes.Buffer
	if err := pageTemplate.ExecuteTemplate(&out, "page.html", p); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(destination, p.Slug+".html"), out.Bytes(), 0o644)
}

func main() {
	title := flag.String("title", "Help", "book title shown in the sidebar and page titles")
	bookID := flag.String("book-id", "", "Apple Help book identifier (optional)")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage: help-book [options] README.md output-directory")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}
	source, err := os.ReadFile(flag.Arg(0))
	if err == nil {
		err = generate(source, flag.Arg(1), *title, *bookID)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "help-book:", err)
		os.Exit(1)
	}
}
