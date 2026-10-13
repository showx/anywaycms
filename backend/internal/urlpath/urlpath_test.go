package urlpath

import "testing"

func TestParseSwitchesSuffix(t *testing.T) {
	htmlArticle := Parse("html", "/article/office-open")
	if !htmlArticle.Redirect || htmlArticle.Path != "/article/office-open.html" || htmlArticle.Slug != "office-open" {
		t.Fatalf("plain to html: %#v", htmlArticle)
	}
	kept := Parse("html", "/article/office-open.html")
	if kept.Redirect || kept.Path != "/article/office-open.html" {
		t.Fatalf("html stays: %#v", kept)
	}
	back := Parse("", "/category/news.HTML")
	if !back.Redirect || back.Kind != "category" || back.Path != "/category/news" || back.Slug != "news" {
		t.Fatalf("html to plain: %#v", back)
	}
	home := Parse("html", "/")
	if home.Kind != "home" || home.Redirect {
		t.Fatalf("home: %#v", home)
	}
	other := Parse("html", "/admin/seo")
	if other.Redirect || other.Kind != "" {
		t.Fatalf("admin: %#v", other)
	}
}
