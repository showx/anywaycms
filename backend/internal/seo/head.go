package seo

import (
	"encoding/json"
	"html"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Head struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
	Robots      string `json:"robots"`
	Canonical   string `json:"canonical"`
	OGType      string `json:"og_type"`
	SiteName    string `json:"site_name"`
	Image       string `json:"image"`
	JSONLD      string `json:"json_ld"`
}

type Entry struct {
	Loc     string
	LastMod string
}

var (
	titleTag = regexp.MustCompile(`(?i)<title>.*?</title>`)
	descTag  = regexp.MustCompile(`(?i)<meta\s+[^>]*name\s*=\s*["']description["'][^>]*>`)
)

func OneLine(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func Compose(pageTitle, siteTitle string) string {
	pageTitle = OneLine(pageTitle)
	siteTitle = OneLine(siteTitle)
	if pageTitle == "" {
		return siteTitle
	}
	if siteTitle == "" || pageTitle == siteTitle {
		return pageTitle
	}
	return pageTitle + " - " + siteTitle
}

func Origin(r *http.Request, canonicalHost string) string {
	scheme := "http"
	if r != nil && r.TLS != nil {
		scheme = "https"
	}
	if r != nil {
		proto := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
		if proto == "http" || proto == "https" {
			scheme = proto
		}
	}
	host := strings.TrimSpace(canonicalHost)
	if r != nil {
		if _, port, err := net.SplitHostPort(r.Host); err == nil && port != "" && port != "80" && port != "443" {
			host = net.JoinHostPort(host, port)
		}
	}
	return scheme + "://" + host
}

func JSONLD(kind string, head Head, published *time.Time) string {
	payload := map[string]any{
		"@context":    "https://schema.org",
		"name":        head.Title,
		"description": head.Description,
		"url":         head.Canonical,
	}
	switch kind {
	case "article":
		payload["@type"] = "Article"
		payload["headline"] = head.Title
		payload["mainEntityOfPage"] = head.Canonical
		if published != nil && !published.IsZero() {
			payload["datePublished"] = published.UTC().Format(time.RFC3339)
		}
		if head.Image != "" {
			payload["image"] = head.Image
		}
	case "category":
		payload["@type"] = "CollectionPage"
	case "page":
		payload["@type"] = "WebPage"
	default:
		payload["@type"] = "WebSite"
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(buf)
}

func (head Head) HTML() string {
	var b strings.Builder
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(head.Title))
	b.WriteString("</title>\n")
	writeName(&b, "description", head.Description)
	writeName(&b, "keywords", head.Keywords)
	writeName(&b, "robots", head.Robots)
	writeProp(&b, "og:title", head.Title)
	writeProp(&b, "og:description", head.Description)
	writeProp(&b, "og:url", head.Canonical)
	writeProp(&b, "og:type", head.OGType)
	writeProp(&b, "og:site_name", head.SiteName)
	if head.Image != "" {
		writeProp(&b, "og:image", head.Image)
	}
	writeName(&b, "twitter:card", "summary")
	writeName(&b, "twitter:title", head.Title)
	writeName(&b, "twitter:description", head.Description)
	if head.Canonical != "" {
		b.WriteString(`<link rel="canonical" href="`)
		b.WriteString(html.EscapeString(head.Canonical))
		b.WriteString("\">\n")
	}
	if head.JSONLD != "" {
		b.WriteString(`<script type="application/ld+json" id="seo-jsonld">`)
		b.WriteString(head.JSONLD)
		b.WriteString("</script>\n")
	}
	return b.String()
}

func Inject(raw []byte, head Head) []byte {
	page := titleTag.ReplaceAllString(string(raw), "")
	page = descTag.ReplaceAllString(page, "")
	block := head.HTML()
	idx := strings.LastIndex(strings.ToLower(page), "</head>")
	if idx < 0 {
		return []byte(block + page)
	}
	return []byte(page[:idx] + block + page[idx:])
}

func RobotsText(indexable bool, sitemapURL string) string {
	if !indexable {
		return "User-agent: *\nDisallow: /\n"
	}
	var b strings.Builder
	b.WriteString("User-agent: *\nAllow: /\nDisallow: /api\n")
	if sitemapURL != "" {
		b.WriteString("\nSitemap: ")
		b.WriteString(sitemapURL)
		b.WriteString("\n")
	}
	return b.String()
}

func SitemapXML(entries []Entry) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString("\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	b.WriteString("\n")
	for _, entry := range entries {
		b.WriteString("  <url><loc>")
		b.WriteString(xmlEscape(entry.Loc))
		b.WriteString("</loc>")
		if entry.LastMod != "" {
			b.WriteString("<lastmod>")
			b.WriteString(xmlEscape(entry.LastMod))
			b.WriteString("</lastmod>")
		}
		b.WriteString("</url>\n")
	}
	b.WriteString("</urlset>\n")
	return b.String()
}

func xmlEscape(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func writeName(b *strings.Builder, name, content string) {
	b.WriteString(`<meta name="`)
	b.WriteString(html.EscapeString(name))
	b.WriteString(`" content="`)
	b.WriteString(html.EscapeString(content))
	b.WriteString("\">\n")
}

func writeProp(b *strings.Builder, name, content string) {
	b.WriteString(`<meta property="`)
	b.WriteString(html.EscapeString(name))
	b.WriteString(`" content="`)
	b.WriteString(html.EscapeString(content))
	b.WriteString("\">\n")
}
