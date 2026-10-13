package router

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"anywaycms/backend/internal/database"
	"anywaycms/backend/internal/model"

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
	return New(db, "test-secret", filepath.Join(dir, "nostatic"), filepath.Join(dir, "uploads")), db
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

func recap(out map[string]any) string {
	buf, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(buf)
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
	if code != http.StatusOK || dataOf(t, out)["system_name"] != "演示后台" || dataOf(t, out)["track_visits"].(float64) != 0 || dataOf(t, out)["admin_path"] == "" {
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

func doRaw(t *testing.T, h http.Handler, target, host string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Host = host
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func TestPublicSEOIsolatedByHost(t *testing.T) {
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
	dist := filepath.Join(dir, "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatal(err)
	}
	index := `<!DOCTYPE html><html><head><title>站群 CMS</title><meta name="description" content="旧描述"></head><body><div id="app"></div></body></html>`
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	engine := New(db, "test-secret", dist, filepath.Join(dir, "uploads"))

	code, body := doRaw(t, engine, "/", "127.0.0.1:8080")
	if code != http.StatusOK || !strings.Contains(body, "<title>星澜科技</title>") || strings.Contains(body, "青禾文旅") || strings.Contains(body, "旧描述") {
		t.Fatalf("home head %d %s", code, body)
	}
	if !strings.Contains(body, `rel="canonical" href="http://localhost:8080/"`) {
		t.Fatalf("canonical host is not primary: %s", body)
	}

	code, body = doRaw(t, engine, "/article/office-open", "www.starlan.local:8080")
	if code != http.StatusOK || !strings.Contains(body, "<title>杭州研发中心启用 - 星澜科技</title>") || !strings.Contains(body, "http://localhost:8080/article/office-open") {
		t.Fatalf("article head %d %s", code, body)
	}
	code, body = doRaw(t, engine, "/article/longjing-walk", "127.0.0.1:8080")
	if code != http.StatusOK || strings.Contains(body, "龙井清晨徒步") || !strings.Contains(body, "noindex,follow") {
		t.Fatalf("cross-site article leaked %d %s", code, body)
	}

	code, body = doRaw(t, engine, "/sitemap.xml", "qinghe.local:8080")
	if code != http.StatusOK || !strings.Contains(body, "/article/longjing-walk") || strings.Contains(body, "office-open") {
		t.Fatalf("qinghe sitemap %d %s", code, body)
	}
	var setting model.Setting
	if err := db.Order("id asc").First(&setting).Error; err != nil {
		t.Fatal(err)
	}
	code, body = doRaw(t, engine, "/robots.txt", "qinghe.local")
	if code != http.StatusOK || strings.Contains(body, "Disallow: /admin") || strings.Contains(body, setting.AdminPath) || !strings.Contains(body, "Sitemap: http://qinghe.local/sitemap.xml") {
		t.Fatalf("robots %d %s", code, body)
	}
	code, body = doRaw(t, engine, "/"+setting.AdminPath+"/login", "127.0.0.1:8080")
	if code != http.StatusOK || !strings.Contains(body, "noindex,nofollow") || !strings.Contains(body, `__ADMIN_BASE__="/`+setting.AdminPath+`"`) || strings.Contains(body, "<title>星澜科技</title>") {
		t.Fatalf("admin head %d %s", code, body)
	}
	code, body = doRaw(t, engine, "/admin/login", "127.0.0.1:8080")
	if strings.Contains(body, "__ADMIN_BASE__") || !strings.Contains(body, "<title>星澜科技</title>") {
		t.Fatalf("fixed admin path leaked %d %s", code, body)
	}
	if strings.Contains(body, setting.AdminPath) {
		t.Fatalf("public page leaked admin path %s", body)
	}

	code, out := doJSON(t, engine, http.MethodPost, "/api/auth/login", "localhost", "", map[string]any{
		"username": "admin",
		"password": "admin123",
	})
	if code != http.StatusOK {
		t.Fatalf("login %d %#v", code, out)
	}
	token := dataOf(t, out)["token"].(string)
	code, out = doJSON(t, engine, http.MethodGet, "/api/system", "127.0.0.1:8080", "", nil)
	if code != http.StatusOK {
		t.Fatalf("public system %d %#v", code, out)
	}
	if _, leaked := dataOf(t, out)["admin_path"]; leaked || strings.Contains(recap(out), setting.AdminPath) {
		t.Fatalf("public system leaked %d %#v", code, out)
	}
	nextPath := "desk8"
	if setting.AdminPath == nextPath {
		nextPath = "desk9"
	}
	code, out = doJSON(t, engine, http.MethodPut, "/api/settings", "localhost", token, map[string]any{
		"system_name": "站群 CMS",
		"admin_path":  "admin",
	})
	if code != http.StatusBadRequest {
		t.Fatalf("rejected path was accepted %d %#v", code, out)
	}
	code, out = doJSON(t, engine, http.MethodPut, "/api/settings", "localhost", token, map[string]any{
		"system_name": "站群 CMS",
		"admin_path":  nextPath,
	})
	if code != http.StatusOK || dataOf(t, out)["admin_path"] != nextPath {
		t.Fatalf("custom path %d %#v", code, out)
	}
	code, body = doRaw(t, engine, "/"+nextPath+"/login", "127.0.0.1:8080")
	if code != http.StatusOK || !strings.Contains(body, `__ADMIN_BASE__="/`+nextPath+`"`) {
		t.Fatalf("new admin head %d %s", code, body)
	}
	code, body = doRaw(t, engine, "/"+setting.AdminPath+"/login", "127.0.0.1:8080")
	if strings.Contains(body, "__ADMIN_BASE__") {
		t.Fatalf("old admin path still injected %s", body)
	}
	code, out = doJSON(t, engine, http.MethodGet, "/api/sites", "localhost", token, nil)
	if code != http.StatusOK {
		t.Fatalf("sites %d %#v", code, out)
	}
	var starlanID float64
	for _, item := range out["data"].([]any) {
		site := item.(map[string]any)
		if site["title"] == "星澜科技" {
			starlanID = site["id"].(float64)
		}
	}
	code, out = doJSON(t, engine, http.MethodGet, "/api/sites/"+itoa(int(starlanID))+"/categories", "localhost", token, nil)
	if code != http.StatusOK {
		t.Fatalf("categories %d %#v", code, out)
	}
	var newsID float64
	for _, item := range out["data"].([]any) {
		cat := item.(map[string]any)
		if cat["slug"] == "news" {
			newsID = cat["id"].(float64)
		}
	}
	code, out = doJSON(t, engine, http.MethodPut, "/api/categories/"+itoa(int(newsID)), "localhost", token, map[string]any{
		"name":            "公司新闻",
		"slug":            "news",
		"description":     "产品发布、客户实践与公司动态。",
		"seo_title":       "新闻中心",
		"seo_description": "星澜的新闻与实践",
	})
	if code != http.StatusOK || dataOf(t, out)["seo_title"] != "新闻中心" {
		t.Fatalf("category seo %d %#v", code, out)
	}
	code, out = doJSON(t, engine, http.MethodGet, "/api/public/seo?path=/category/news", "starlan.local:8080", "", nil)
	if code != http.StatusOK || dataOf(t, out)["title"] != "新闻中心 - 星澜科技" || !strings.Contains(dataOf(t, out)["canonical"].(string), "http://localhost:8080/category/news") {
		t.Fatalf("public seo %d %#v", code, out)
	}

	code, out = doJSON(t, engine, http.MethodPut, "/api/sites/"+itoa(int(starlanID))+"/seo", "127.0.0.1:8080", token, map[string]any{
		"seo_title":   "星澜官网",
		"description": "星澜科技帮助制造和零售企业把订单、库存和生产连成一条可追踪的数据链路。",
		"keywords":    "星澜科技,MES",
		"robots":      "noindex",
	})
	if code != http.StatusOK || dataOf(t, out)["robots"] != "noindex" || dataOf(t, out)["seo_title"] != "星澜官网" {
		t.Fatalf("update seo %d %#v", code, out)
	}
	code, body = doRaw(t, engine, "/robots.txt", "127.0.0.1:8080")
	if code != http.StatusOK || !strings.Contains(body, "Disallow: /") || strings.Contains(body, "Sitemap:") {
		t.Fatalf("noindex robots %d %s", code, body)
	}
	code, body = doRaw(t, engine, "/", "starlan.local:8080")
	if code != http.StatusOK || !strings.Contains(body, "<title>星澜官网</title>") || !strings.Contains(body, "noindex,nofollow") {
		t.Fatalf("home after seo save %d %s", code, body)
	}
	code, body = doRaw(t, engine, "/", "qinghe.local:8080")
	if code != http.StatusOK || !strings.Contains(body, "<title>青禾文旅</title>") || strings.Contains(body, "noindex,nofollow") {
		t.Fatalf("other site affected %d %s", code, body)
	}
}

func TestPseudoStaticIsPerSite(t *testing.T) {
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
	dist := filepath.Join(dir, "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatal(err)
	}
	index := `<!DOCTYPE html><html><head><title>站群 CMS</title></head><body></body></html>`
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	engine := New(db, "test-secret", dist, filepath.Join(dir, "uploads"))
	code, out := doJSON(t, engine, http.MethodPost, "/api/auth/login", "localhost", "", map[string]any{
		"username": "admin",
		"password": "admin123",
	})
	if code != http.StatusOK {
		t.Fatalf("login %d %#v", code, out)
	}
	token := dataOf(t, out)["token"].(string)
	code, out = doJSON(t, engine, http.MethodGet, "/api/sites", "localhost", token, nil)
	if code != http.StatusOK {
		t.Fatalf("sites %d %#v", code, out)
	}
	var starlanID float64
	for _, item := range out["data"].([]any) {
		site := item.(map[string]any)
		if site["title"] == "星澜科技" {
			starlanID = site["id"].(float64)
		}
	}
	code, out = doJSON(t, engine, http.MethodPut, "/api/sites/"+itoa(int(starlanID))+"/seo", "localhost", token, map[string]any{
		"description": "星澜科技帮助制造和零售企业把订单、库存和生产连成一条可追踪的数据链路。",
		"keywords":    "星澜科技,MES,企业数字化",
		"robots":      "index",
		"url_mode":    "html",
	})
	if code != http.StatusOK || dataOf(t, out)["url_mode"] != "html" {
		t.Fatalf("enable html %d %#v", code, out)
	}
	locs, ok := dataOf(t, out)["urls"].([]any)
	if !ok || !hasLoc(locs, "http://localhost/article/office-open.html") || hasLoc(locs, "http://localhost/article/longjing-walk.html") {
		t.Fatalf("sitemap urls %#v", dataOf(t, out)["urls"])
	}

	status, loc, _ := doHeaders(t, engine, "/article/office-open?page=2", "starlan.local:8080")
	if status != http.StatusMovedPermanently || loc != "/article/office-open.html?page=2" {
		t.Fatalf("redirect %d %s", status, loc)
	}
	status, body := doRaw(t, engine, "/article/office-open.html", "www.starlan.local:8080")
	if status != http.StatusOK || !strings.Contains(body, "http://localhost:8080/article/office-open.html") {
		t.Fatalf("html head %d %s", status, body)
	}
	status, loc, _ = doHeaders(t, engine, "/article/longjing-walk.html", "qinghe.local")
	if status != http.StatusMovedPermanently || loc != "/article/longjing-walk" {
		t.Fatalf("qinghe stays plain %d %s", status, loc)
	}
	code, out = doJSON(t, engine, http.MethodGet, "/api/public/bootstrap", "starlan.local", "", nil)
	if code != http.StatusOK {
		t.Fatalf("bootstrap %d %#v", code, out)
	}
	menus := dataOf(t, out)["menus"].([]any)
	if !menuHasSuffix(menus, ".html") {
		t.Fatalf("menus not rewritten: %#v", menus)
	}
}

func TestAssetsStayOnTheirSite(t *testing.T) {
	engine, _ := newEngine(t)
	code, out := doJSON(t, engine, http.MethodPost, "/api/auth/login", "localhost", "", map[string]any{
		"username": "admin",
		"password": "admin123",
	})
	if code != http.StatusOK {
		t.Fatalf("login %d %#v", code, out)
	}
	token := dataOf(t, out)["token"].(string)
	code, out = doJSON(t, engine, http.MethodGet, "/api/sites", "localhost", token, nil)
	if code != http.StatusOK {
		t.Fatalf("sites %d %#v", code, out)
	}
	var starlanID, qingheID float64
	for _, item := range out["data"].([]any) {
		site := item.(map[string]any)
		switch site["title"] {
		case "星澜科技":
			starlanID = site["id"].(float64)
		case "青禾文旅":
			qingheID = site["id"].(float64)
		}
	}

	code, out = postFile(t, engine, "/api/sites/"+itoa(int(starlanID))+"/assets", token, "photo.png", tinyPNG(t))
	if code != http.StatusOK {
		t.Fatalf("upload %d %#v", code, out)
	}
	uploaded := dataOf(t, out)
	url, _ := uploaded["url"].(string)
	if uploaded["mime"] != "image/png" || !strings.HasPrefix(url, "/uploads/") || !strings.HasSuffix(url, ".png") {
		t.Fatalf("asset %#v", uploaded)
	}
	status, body := doRaw(t, engine, url, "127.0.0.1:8080")
	if status != http.StatusOK || !strings.HasPrefix(body, "\x89PNG") {
		t.Fatalf("serve %d %q", status, body[:min(20, len(body))])
	}

	code, out = doJSON(t, engine, http.MethodGet, "/api/sites/"+itoa(int(qingheID))+"/assets", "localhost", token, nil)
	if code != http.StatusOK || len(out["data"].([]any)) != 0 {
		t.Fatalf("qinghe saw starlan file %d %#v", code, out)
	}
	code, out = postFile(t, engine, "/api/sites/"+itoa(int(starlanID))+"/assets", token, "note.html", []byte("<html>no</html>"))
	if code != http.StatusBadRequest {
		t.Fatalf("html upload %d %#v", code, out)
	}
	code, out = doJSON(t, engine, http.MethodDelete, "/api/assets/"+itoa(int(uploaded["id"].(float64))), "localhost", token, nil)
	if code != http.StatusOK {
		t.Fatalf("delete %d %#v", code, out)
	}
	status, _ = doRaw(t, engine, url, "127.0.0.1:8080")
	if status != http.StatusNotFound {
		t.Fatalf("deleted file still served %d", status)
	}
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func postFile(t *testing.T, h http.Handler, target, token, name string, payload []byte) (int, map[string]any) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Host = "localhost"
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
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

func doHeaders(t *testing.T, h http.Handler, target, host string) (int, string, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Host = host
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Header().Get("Location"), rec.Body.String()
}

func hasLoc(list []any, loc string) bool {
	for _, item := range list {
		if item.(map[string]any)["loc"] == loc {
			return true
		}
	}
	return false
}

func menuHasSuffix(list []any, suffix string) bool {
	for _, item := range list {
		url, _ := item.(map[string]any)["url"].(string)
		if strings.HasSuffix(url, suffix) {
			return true
		}
	}
	return false
}
