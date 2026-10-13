package hostutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"LocalHost:8080":             "localhost",
		"https://Qinghe.local/a?x=1": "qinghe.local",
		"127.0.0.1:8080":             "127.0.0.1",
		"WWW.Starlan.local.":         "www.starlan.local",
		"  Demo.Local ":              "demo.local",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Fatalf("%q => %q, want %q", in, got, want)
		}
	}
	if !Valid("qinghe.local") || Valid("bad host") {
		t.Fatal("valid check failed")
	}
}

func TestFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/?host=Qinghe.local", nil)
	req.Host = "localhost:8080"
	if got := FromRequest(req); got != "qinghe.local" {
		t.Fatalf("query host = %s", got)
	}
	req.Header.Set("X-Site-Host", "Starlan.local:5173")
	if got := FromRequest(req); got != "starlan.local" {
		t.Fatalf("header host = %s", got)
	}
}
