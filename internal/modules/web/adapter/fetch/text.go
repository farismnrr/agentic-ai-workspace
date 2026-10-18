package fetch

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

// skipped are elements whose text is not the page's content: code, menus, forms and other furniture.
var skipped = map[string]bool{
	"script": true, "style": true, "noscript": true, "template": true, "svg": true, "iframe": true, "nav": true,
	"footer": true, "aside": true, "form": true, "button": true, "select": true,
}

// breaks are elements that start a new line of text.
var breaks = map[string]bool{
	"p": true, "div": true, "br": true, "li": true, "tr": true, "section": true, "article": true, "main": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "blockquote": true, "pre": true,
	"table": true, "ul": true, "ol": true, "header": true, "figcaption": true,
}

// lineGuess is the usual number of text blocks on a page, used to size the buffer.
const lineGuess = 64

// extract reads HTML into its title and the text a person would read, one block per line, with no markup and no
// scripts. A page that cannot be parsed yields what could be read.
func extract(r io.Reader) (string, string) {
	doc, err := html.Parse(r)
	if err != nil && doc == nil {
		return "", ""
	}

	var out strings.Builder

	title := ""
	write(&out, doc, &title)

	return title, tidy(out.String())
}

// write adds the readable text of a node to out. The head only gives the title; skipped elements give nothing.
func write(out *strings.Builder, n *html.Node, title *string) {
	if n.Type == html.ElementNode {
		if n.Data == "head" {
			*title = titleOf(n)

			return
		}

		if skipped[n.Data] {
			return
		}
	}

	block := n.Type == html.ElementNode && breaks[n.Data]
	if block {
		out.WriteString("\n")
	}

	if n.Type == html.TextNode {
		out.WriteString(n.Data + " ")
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		write(out, child, title)
	}

	if block {
		out.WriteString("\n")
	}
}

// titleOf finds the page title inside the head, without reading anything else there.
func titleOf(head *html.Node) string {
	for child := head.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "title" && child.FirstChild != nil {
			return strings.Join(strings.Fields(child.FirstChild.Data), " ")
		}
	}

	return ""
}

// tidy collapses spaces inside each line and drops empty lines and repeated blank ones.
func tidy(raw string) string {
	lines := make([]string, 0, lineGuess)

	for _, line := range strings.Split(raw, "\n") {
		if line = strings.Join(strings.Fields(line), " "); line != "" {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}
