package handler

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanRelRejectsEscape(t *testing.T) {
	if _, err := cleanRel("../secret.html"); err == nil {
		t.Fatal("expected rejection")
	}
	if _, err := cleanRel("css/../../secret.html"); err == nil {
		t.Fatal("expected nested rejection")
	}
	if _, err := cleanRel("shell.php"); err == nil {
		t.Fatal("expected php rejection")
	}
	got, err := cleanRel(`.\index.html`)
	if err != nil || got != "index.html" {
		t.Fatalf("clean = %q %v", got, err)
	}
}

func TestCandidateRels(t *testing.T) {
	got := strings.Join(candidateRels("/"), ",")
	if got != "index.html,index.htm" {
		t.Fatalf("root = %s", got)
	}
	got = strings.Join(candidateRels("/about"), ",")
	if got != "about.html,about.htm,about/index.html,about/index.htm" {
		t.Fatalf("about = %s", got)
	}
	got = strings.Join(candidateRels("/css/a.css"), ",")
	if got != "css/a.css" {
		t.Fatalf("css = %s", got)
	}
	got = strings.Join(candidateRels("/../secret.html"), ",")
	if got != "secret.html" {
		t.Fatalf("normalized = %s", got)
	}
	if candidateRels("/.env") != nil {
		t.Fatal("hidden file should have no candidate")
	}
}

func TestRewritePreviewKeepsRemoteLinks(t *testing.T) {
	body := `<a href="css/a.css">x</a><a href="https://example.com/a">y</a><a href="#top">z</a>`
	out := rewritePreviewRefs(".html", body, "demo.local")
	if !strings.Contains(out, `href="css/a.css?host=demo.local"`) {
		t.Fatalf("relative not rewritten: %s", out)
	}
	if !strings.Contains(out, `href="https://example.com/a"`) || !strings.Contains(out, `href="#top"`) {
		t.Fatalf("remote link changed: %s", out)
	}
	css := rewritePreviewRefs(".css", `p{background:url(../a.png)}`, "demo.local")
	if !strings.Contains(css, "a.png?host=demo.local") {
		t.Fatalf("css = %s", css)
	}
}

func TestExtractZipBlocksSlip(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "site")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := extractZip(zipBytes(t, map[string]string{
		"index.html":     "ok",
		"../outside.txt": "no",
	}), dest)
	if err == nil {
		t.Fatal("expected slip rejection")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "outside.txt")); !os.IsNotExist(statErr) {
		t.Fatal("file escaped the site directory")
	}
}

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
