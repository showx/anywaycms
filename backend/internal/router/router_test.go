package router

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"anywaycms/backend/internal/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func newEngine(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "cms.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	})
	return New(db, "test-secret", filepath.Join(dir, "nostatic")), db
}

func doJSON(t *testing.T, h http.Handler, method, target, host, token string, payload any) (int, map[string]any) {
	t.Helper()
	var body io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(buf)
	}
	req := httptest.NewRequest(method, target, body)
	req.Host = host
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
	}
	return rec.Code, out
}

func dataOf(t *testing.T, out map[string]any) map[string]any {
	t.Helper()
	data, ok := out["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data: %#v", out)
	}
	return data
}

func TestTwoDomainsServeDifferentPages(t *testing.T) {
	engine, _ := newEngine(t)

	code, out := doJSON(t, engine, http.MethodGet, "/api/public/bootstrap", "starlan.local", "", nil)
	if code != http.StatusOK {
		t.Fatalf("starlan bootstrap %d %#v", code, out)
	}
	if dataOf(t, out)["site"].(map[string]any)["title"] != "星澜科技" {
		t.Fatalf("starlan site = %#v", out)
	}

	code, out = doJSON(t, engine, http.MethodGet, "/api/public/bootstrap", "www.qinghe.local:443", "", nil)
	if code != http.StatusOK {
		t.Fatalf("qinghe bootstrap %d %#v", code, out)
	}
	if dataOf(t, out)["site"].(map[string]any)["title"] != "青禾文旅" {
		t.Fatalf("qinghe site = %#v", out)
	}

	code, out = doJSON(t, engine, http.MethodGet, "/api/public/pages/about", "localhost:8080", "", nil)
	if code != http.StatusOK || !strings.Contains(dataOf(t, out)["body"].(string), "星澜科技") {
		t.Fatalf("localhost about = %d %#v", code, out)
	}
	code, out = doJSON(t, engine, http.MethodGet, "/api/public/pages/about", "qinghe.local", "", nil)
	body := ""
	if code == http.StatusOK {
		body = dataOf(t, out)["body"].(string)
	}
	if code != http.StatusOK || !strings.Contains(body, "青禾文旅") || strings.Contains(body, "星澜科技") {
		t.Fatalf("qinghe about leaked or missing: %d %s", code, body)
	}

	code, _ = doJSON(t, engine, http.MethodGet, "/api/public/categories/news", "qinghe.local", "", nil)
	if code != http.StatusNotFound {
		t.Fatalf("qinghe should not see starlan news, got %d", code)
	}
	code, out = doJSON(t, engine, http.MethodGet, "/api/public/categories/news", "starlan.local", "", nil)
	if code != http.StatusOK || dataOf(t, out)["category"].(map[string]any)["name"] != "公司新闻" {
		t.Fatalf("starlan news = %d %#v", code, out)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/public/bootstrap", nil)
	req.Host = "starlan.local"
	req.Header.Set("X-Site-Host", "qinghe.local")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "青禾文旅") {
		t.Fatalf("preview header failed: %d %s", rec.Code, rec.Body.String())
	}

	code, out = doJSON(t, engine, http.MethodGet, "/api/public/bootstrap", "missing.example", "", nil)
	if code != http.StatusNotFound {
		t.Fatalf("unbound domain = %d %#v", code, out)
	}
}

func TestAdminCanBindAnotherDomain(t *testing.T) {
	engine, _ := newEngine(t)
	code, _ := doJSON(t, engine, http.MethodGet, "/api/sites", "localhost", "", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("sites without token = %d", code)
	}

	code, out := doJSON(t, engine, http.MethodPost, "/api/auth/login", "localhost", "", map[string]any{
		"username": "admin",
		"password": "admin123",
	})
	if code != http.StatusOK {
		t.Fatalf("login %d %#v", code, out)
	}
	token := dataOf(t, out)["token"].(string)

	code, out = doJSON(t, engine, http.MethodPost, "/api/sites", "localhost", token, map[string]any{
		"name":   "测试站",
		"title":  "测试站标题",
		"theme":  `{"layout":"magazine","primary":"#112233"}`,
		"status": 1,
		"hosts":  []string{"https://Demo.local/path", "www.demo.local"},
	})
	if code != http.StatusOK {
		t.Fatalf("create site %d %#v", code, out)
	}
	created := dataOf(t, out)
	if created["title"] != "测试站标题" {
		t.Fatalf("created = %#v", created)
	}
	domains := created["domains"].([]any)
	if len(domains) != 2 {
		t.Fatalf("domains = %#v", domains)
	}
	first := domains[0].(map[string]any)
	second := domains[1].(map[string]any)
	if first["host"] != "demo.local" || first["is_primary"] != true {
		t.Fatalf("primary domain = %#v", first)
	}
	if second["host"] != "www.demo.local" || second["is_primary"] != false {
		t.Fatalf("alias domain = %#v", second)
	}

	code, out = doJSON(t, engine, http.MethodGet, "/api/public/bootstrap", "demo.local", "", nil)
	if code != http.StatusOK || dataOf(t, out)["site"].(map[string]any)["title"] != "测试站标题" {
		t.Fatalf("demo bootstrap %d %#v", code, out)
	}
	code, out = doJSON(t, engine, http.MethodGet, "/api/public/bootstrap", "starlan.local", "", nil)
	if code != http.StatusOK || dataOf(t, out)["site"].(map[string]any)["title"] != "星澜科技" {
		t.Fatalf("starlan changed %d %#v", code, out)
	}

	code, out = doJSON(t, engine, http.MethodPost, "/api/sites", "localhost", token, map[string]any{
		"name":  "冲突站",
		"title": "冲突站",
		"hosts": []string{"localhost"},
	})
	if code != http.StatusBadRequest {
		t.Fatalf("duplicate domain = %d %#v", code, out)
	}
}

func TestVisitStatsStayOnTheirSite(t *testing.T) {
	engine, _ := newEngine(t)
	code, out := doJSON(t, engine, http.MethodPost, "/api/auth/login", "localhost", "", map[string]any{
		"username": "admin",
		"password": "admin123",
	})
	if code != http.StatusOK {
		t.Fatalf("login %d %#v", code, out)
	}
	token := dataOf(t, out)["token"].(string)

	code, out = doJSON(t, engine, http.MethodGet, "/api/system", "localhost", "", nil)
	if code != http.StatusOK || dataOf(t, out)["system_name"] != "站群 CMS" {
		t.Fatalf("system %d %#v", code, out)
	}

	code, out = doJSON(t, engine, http.MethodGet, "/api/sites", "localhost", token, nil)
	if code != http.StatusOK {
		t.Fatalf("sites %d %#v", code, out)
	}
	sites := out["data"].([]any)
	var starlanID, qingheID float64
	for _, item := range sites {
		site := item.(map[string]any)
		switch site["title"] {
		case "星澜科技":
			starlanID = site["id"].(float64)
		case "青禾文旅":
			qingheID = site["id"].(float64)
		}
	}
	if starlanID == 0 || qingheID == 0 {
		t.Fatalf("missing demo sites: %#v", sites)
	}

	code, out = doJSON(t, engine, http.MethodPost, "/api/public/visit", "starlan.local", "", map[string]any{"path": "/page/about"})
	if code != http.StatusOK || dataOf(t, out)["recorded"] != true {
		t.Fatalf("record starlan %d %#v", code, out)
	}
	code, out = doJSON(t, engine, http.MethodPost, "/api/public/visit", "starlan.local", "", map[string]any{"path": "/"})
	if code != http.StatusOK {
		t.Fatalf("record home %d %#v", code, out)
	}
	code, out = doJSON(t, engine, http.MethodPost, "/api/public/visit", "qinghe.local", "", map[string]any{"path": "/category/notes"})
	if code != http.StatusOK {
		t.Fatalf("record qinghe %d %#v", code, out)
	}

	code, out = doJSON(t, engine, http.MethodGet, "/api/stats?days=7&site_id="+itoa(int(starlanID)), "localhost", token, nil)
	if code != http.StatusOK {
		t.Fatalf("starlan stats %d %#v", code, out)
	}
	summary := dataOf(t, out)["summary"].(map[string]any)
	if summary["views"].(float64) != 2 || summary["visitors"].(float64) != 1 {
		t.Fatalf("starlan summary = %#v", summary)
	}
	if !hasNamed(dataOf(t, out)["paths"], "/page/about") || hasNamed(dataOf(t, out)["paths"], "/category/notes") {
		t.Fatalf("starlan paths leaked: %#v", dataOf(t, out)["paths"])
	}

	code, out = doJSON(t, engine, http.MethodGet, "/api/stats?days=7&site_id="+itoa(int(qingheID)), "localhost", token, nil)
	if code != http.StatusOK || dataOf(t, out)["summary"].(map[string]any)["views"].(float64) != 1 {
		t.Fatalf("qinghe stats %d %#v", code, out)
	}

	code, out = doJSON(t, engine, http.MethodPut, "/api/settings", "localhost", token, map[string]any{
		"system_name":   "演示后台",
		"system_notice": "只做统计开关测试",
		"track_visits":  0,
		"retain_days":   30,
	})
	if code != http.StatusOK || dataOf(t, out)["system_name"] != "演示后台" || dataOf(t, out)["track_visits"].(float64) != 0 {
		t.Fatalf("update settings %d %#v", code, out)
	}
	code, out = doJSON(t, engine, http.MethodPost, "/api/public/visit", "qinghe.local", "", map[string]any{"path": "/"})
	if code != http.StatusOK || dataOf(t, out)["recorded"] != false {
		t.Fatalf("tracking off %d %#v", code, out)
	}
	code, out = doJSON(t, engine, http.MethodGet, "/api/stats?days=7&site_id="+itoa(int(qingheID)), "localhost", token, nil)
	if code != http.StatusOK || dataOf(t, out)["summary"].(map[string]any)["views"].(float64) != 1 {
		t.Fatalf("count changed while tracking off %d %#v", code, out)
	}
}

func hasNamed(list any, name string) bool {
	items, ok := list.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		if item.(map[string]any)["name"] == name {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
