package desktop

import (
	"strings"

	"golang.org/x/net/html"
)

// RichSpan is a safe, structured text span extracted from HTML fragments (`<strong>`, `<i>`, `<em>`)
// so the React frontend can render rich typography without ever using dangerouslySetInnerHTML.
type RichSpan struct {
	Text   string `json:"text"`
	Bold   bool   `json:"bold,omitempty"`
	Italic bool   `json:"italic,omitempty"`
}

// ParseSafeRichSpans parses an HTML fragment and extracts plain text + safe Bold/Italic spans,
// stripping all scripts, styles, event handlers, and unsafe tags.
func ParseSafeRichSpans(rawHTML string) (plainText string, spans []RichSpan) {
	trimmed := strings.TrimSpace(rawHTML)
	if trimmed == "" {
		return "", nil
	}
	doc, err := html.Parse(strings.NewReader(trimmed))
	if err != nil {
		return trimmed, []RichSpan{{Text: trimmed}}
	}

	var rawSpans []RichSpan
	var walk func(n *html.Node, bold, italic bool)
	walk = func(n *html.Node, bold, italic bool) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			if tag == "script" || tag == "style" || tag == "noscript" {
				return
			}
			switch tag {
			case "strong", "b":
				bold = true
			case "em", "i", "dfn":
				italic = true
			}
		}
		if n.Type == html.TextNode {
			txt := n.Data
			if txt != "" {
				rawSpans = append(rawSpans, RichSpan{
					Text:   txt,
					Bold:   bold,
					Italic: italic,
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, bold, italic)
		}
	}
	walk(doc, false, false)

	// Coalesce adjacent spans with identical formatting
	var merged []RichSpan
	var plainBuilder strings.Builder
	for _, sp := range rawSpans {
		plainBuilder.WriteString(sp.Text)
		if len(merged) > 0 && merged[len(merged)-1].Bold == sp.Bold && merged[len(merged)-1].Italic == sp.Italic {
			merged[len(merged)-1].Text += sp.Text
		} else {
			merged = append(merged, sp)
		}
	}
	plain := strings.Join(strings.Fields(plainBuilder.String()), " ")
	return plain, merged
}
