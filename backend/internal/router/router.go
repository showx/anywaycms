package router

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"anywaycms/backend/internal/handler"
	"anywaycms/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const devHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <title>站群 CMS</title>
  <style>
    body{font-family:"Segoe UI","Microsoft YaHei",sans-serif;background:#eef1f4;color:#1c2430;display:grid;min-height:100vh;place-items:center;margin:0}
    main{max-width:560px;background:#fff;padding:28px;border-radius:16px;line-height:1.7}
    code{background:#f4f0e4;padding:2px 6px;border-radius:6px}
  </style>
</head>
<body>
  <main>
    <h1>前端还没有构建</h1>
    <p>接口已经可用。构建前端后再刷新本页：</p>
    <p><code>cd frontend</code><br><code>npm install</code><br><code>npm run build</code></p>
    <p>开发时也可以另开终端执行 <code>npm run dev</code>，然后打开 <code>http://127.0.0.1:5173</code>。</p>
  </main>
</body>
</html>`

func New(db *gorm.DB, secret, staticDir, uploadDir string) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	if gin.Mode() != gin.TestMode {
		engine.Use(gin.Logger())
	}
	engine.Use(middleware.CORS(), middleware.APIHeaders())

	h := handler.New(db, secret, uploadDir)
	api := engine.Group("/api")
	api.GET("/health", h.Health)
	api.GET("/system", h.PublicSystem)
	api.POST("/auth/login", h.Login)

	pub := api.Group("/public", middleware.ResolveSite(db))
	pub.GET("/bootstrap", h.Bootstrap)
	pub.POST("/visit", h.RecordVisit)
	pub.GET("/seo", h.PublicSEO)
	pub.GET("/home", h.Home)
	pub.GET("/categories/:slug", h.PublicCategory)
	pub.GET("/articles/:slug", h.PublicArticle)
	pub.GET("/pages/:slug", h.PublicPage)

	auth := api.Group("", middleware.JWT(secret))
	auth.GET("/auth/me", h.Me)
	auth.PUT("/auth/password", h.Password)
	auth.GET("/settings", h.GetSettings)
	auth.PUT("/settings", h.UpdateSettings)
	auth.GET("/stats", h.Stats)
	auth.GET("/sites/:id/seo", h.GetSEO)
	auth.PUT("/sites/:id/seo", h.UpdateSEO)
	auth.GET("/sites", h.ListSites)
	auth.POST("/sites", h.CreateSite)
	auth.GET("/sites/:id", h.GetSite)
	auth.PUT("/sites/:id", h.UpdateSite)
	auth.DELETE("/sites/:id", h.DeleteSite)
	auth.POST("/sites/:id/domains", h.CreateDomain)
	auth.PUT("/domains/:id/primary", h.PrimaryDomain)
	auth.DELETE("/domains/:id", h.DeleteDomain)
	auth.GET("/sites/:id/categories", h.ListCategories)
	auth.POST("/sites/:id/categories", h.CreateCategory)
	auth.PUT("/categories/:id", h.UpdateCategory)
	auth.DELETE("/categories/:id", h.DeleteCategory)
	auth.GET("/sites/:id/contents", h.ListContents)
	auth.POST("/sites/:id/contents", h.CreateContent)
	auth.GET("/contents/:id", h.GetContent)
	auth.PUT("/contents/:id", h.UpdateContent)
	auth.DELETE("/contents/:id", h.DeleteContent)
	auth.GET("/sites/:id/assets", h.ListAssets)
	auth.POST("/sites/:id/assets", h.UploadAsset)
	auth.DELETE("/assets/:id", h.DeleteAsset)
	auth.GET("/sites/:id/menus", h.ListMenus)
	auth.POST("/sites/:id/menus", h.CreateMenu)
	auth.PUT("/menus/:id", h.UpdateMenu)
	auth.DELETE("/menus/:id", h.DeleteMenu)

	engine.GET("/uploads/:site/:name", h.ServeAsset)
	engine.GET("/robots.txt", middleware.ResolveSite(db), h.Robots)
	engine.GET("/sitemap.xml", middleware.ResolveSite(db), h.Sitemap)

	if indexExists(staticDir) {
		engine.NoRoute(spa(staticDir, h))
	} else {
		engine.NoRoute(func(c *gin.Context) {
			if isAPI(c.Request.URL.Path) {
				c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "接口不存在"})
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(devHTML))
		})
	}
	return engine
}

func indexExists(dir string) bool {
	if strings.TrimSpace(dir) == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, "index.html"))
	return err == nil && !info.IsDir()
}

func isAPI(urlPath string) bool {
	clean := path.Clean("/" + urlPath)
	return clean == "/api" || strings.HasPrefix(clean, "/api/")
}

func spa(dir string, h *handler.Handler) gin.HandlerFunc {
	root, err := filepath.Abs(dir)
	if err != nil {
		root = dir
	}
	index := filepath.Join(root, "index.html")
	return func(c *gin.Context) {
		if isAPI(c.Request.URL.Path) {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "接口不存在"})
			return
		}
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}
		urlPath := path.Clean("/" + c.Request.URL.Path)
		rel := strings.TrimPrefix(urlPath, "/")
		full := filepath.Join(root, filepath.FromSlash(rel))
		abs, err := filepath.Abs(full)
		if err != nil || (abs != root && !strings.HasPrefix(abs, root+string(os.PathSeparator))) {
			c.Status(http.StatusForbidden)
			return
		}
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			c.File(abs)
			return
		}
		if urlPath == "/index.html" {
			urlPath = "/"
		}
		if loc := h.PublicRedirect(c.Request, urlPath); loc != "" {
			c.Redirect(http.StatusMovedPermanently, loc)
			return
		}
		raw, err := os.ReadFile(index)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Header("Cache-Control", "no-cache")
		var body []byte
		if h.IsAdminPath(urlPath) {
			body = h.InjectAdmin(raw)
		} else {
			body = h.InjectPublic(c.Request, raw, urlPath)
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", body)
	}
}
