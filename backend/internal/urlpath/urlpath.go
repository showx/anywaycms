package urlpath

import (
	"net/url"
	"path"
	"strings"
)

var kinds = []string{"category", "article", "page"}

func Stored(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "html") {
		return "html"
	}
	return ""
}

func Label(stored string) string {
	if stored == "html" {
		return "html"
	}
	return "plain"
}

func Clean(raw string) string {
	raw = strings.TrimSpace(raw)
	if cut := strings.IndexAny(raw, "?#"); cut >= 0 {
		raw = raw[:cut]
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return "/"
	}
	cleaned := path.Clean("/" + decoded)
	if cleaned == "." || strings.Contains(cleaned, "..") {
		return "/"
	}
	return cleaned
}

func Join(stored, kind, slug string) string {
	target := "/" + kind + "/" + slug
	if stored == "html" {
		return target + ".html"
	}
	return target
}

type Match struct {
	Kind     string
	Slug     string
	Path     string
	Redirect bool
}

func Parse(stored, raw string) Match {
	cleaned := Clean(raw)
	if cleaned == "/" {
		return Match{Kind: "home", Path: "/"}
	}
	for _, kind := range kinds {
		prefix := "/" + kind + "/"
		if !strings.HasPrefix(cleaned, prefix) {
			continue
		}
		rest := strings.TrimPrefix(cleaned, prefix)
		if rest == "" || strings.Contains(rest, "/") {
			return Match{Path: cleaned}
		}
		slug := rest
		if strings.HasSuffix(strings.ToLower(slug), ".html") {
			slug = slug[:len(slug)-len(".html")]
		}
		if slug == "" {
			return Match{Path: cleaned}
		}
		preferred := Join(stored, kind, slug)
		return Match{
			Kind:     kind,
			Slug:     slug,
			Path:     preferred,
			Redirect: preferred != cleaned,
		}
	}
	return Match{Path: cleaned}
}
