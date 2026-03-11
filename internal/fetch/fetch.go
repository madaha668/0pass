package fetch

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// PageInfo holds metadata extracted from a web page.
type PageInfo struct {
	Title       string
	Description string
}

// FetchPageInfo fetches a URL and extracts the page title and meta description.
// If the URL has no scheme, "https://" is prepended.
// Partial results are returned even if some fields could not be extracted.
func FetchPageInfo(rawURL string) (*PageInfo, error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetching page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	info := &PageInfo{}
	extractInfo(doc, info)
	return info, nil
}

func extractInfo(n *html.Node, info *PageInfo) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "title":
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				info.Title = strings.TrimSpace(n.FirstChild.Data)
			}
		case "meta":
			var name, content string
			for _, attr := range n.Attr {
				switch strings.ToLower(attr.Key) {
				case "name":
					name = strings.ToLower(attr.Val)
				case "content":
					content = attr.Val
				}
			}
			if name == "description" && content != "" {
				info.Description = strings.TrimSpace(content)
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractInfo(c, info)
	}
}
