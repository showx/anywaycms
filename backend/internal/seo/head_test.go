package seo

import (
	"net/http"
	"strings"
	"testing"
)

func TestComposeAndInject(t *testing.T) {
	if got := Compose("研发中心", "星澜科技"); got != "研发中心 - 星澜科技" {
		t.Fatalf("compose = %q", got)
	}
	if got := Compose("星澜科技", "星澜科技"); got != "星澜科技" {
		t.Fatalf("same title = %q", got)
	}
	raw := []byte(`<html><head><title>旧标题</title><meta name="description" content="旧描述"></head><body></body></html>`)
	out := string(Inject(raw, Head{
		Title:       `研发 & 中心`,
		Description: "分开部署",
		Robots:      "index,follow",
		Canonical:   "http://localhost:8080/article/a",
		OGType:      "article",
		SiteName:    "星澜科技",
		JSONLD:      `{"name":"研发 & 中心"}`,
	}))
	if strings.Contains(out, "旧标题") || strings.Contains(out, "旧描述") {
		t.Fatalf("old tags kept: %s", out)
	}
	if !strings.Contains(out, "<title>研发 &amp; 中心</title>") {
		t.Fatalf("title not escaped: %s", out)
	}
	if !strings.Contains(out, `rel="canonical" href="http://localhost:8080/article/a"`) {
		t.Fatalf("canonical missing: %s", out)
	}
}

func TestRobotsAndOrigin(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080/", nil)
	req.Host = "127.0.0.1:8080"
	req.Header.Set("X-Forwarded-Proto", "https")
	if got := Origin(req, "localhost"); got != "https://localhost:8080" {
		t.Fatalf("origin = %q", got)
	}
	text := RobotsText(true, "https://localhost:8080/sitemap.xml")
	if !strings.Contains(text, "Disallow: /api") || strings.Contains(text, "Disallow: /admin") || !strings.Contains(text, "Sitemap: https://localhost:8080/sitemap.xml") {
		t.Fatalf("robots = %q", text)
	}
	if RobotsText(false, "") != "User-agent: *\nDisallow: /\n" {
		t.Fatal("noindex robots mismatch")
	}
	xml := SitemapXML([]Entry{{Loc: "http://localhost/a?b=1&c=2", LastMod: "2026-10-13"}})
	if !strings.Contains(xml, "a?b=1&amp;c=2") {
		t.Fatalf("sitemap = %s", xml)
	}
}
